package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/opensearchservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"
	"github.com/hashicorp/go-multierror"
)

// getAll queries AWS for all active domains in the account that meet the nuking criteria based on
// the excludeAfter and configObj configurations. Note that OpenSearch Domains do not have resource timestamps, so we
// use the first-seen tagging pattern to track which OpenSearch Domains should be nuked based on time. This routine will
// tag resources with the first-seen tag if it does not have one.
func (osd *OpenSearchDomains) getAll(configObj config.Config) ([]*string, error) {
	domains, err := osd.getAllActiveOpenSearchDomains()
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	domainsToNuke := []*string{}
	for _, domain := range domains {

		firstSeenTime, err := osd.getFirstSeenTag(domain.ARN)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime.IsZero() {
			err := osd.setFirstSeenTag(domain.ARN, time.Now().UTC())
			if err != nil {
				logging.Logger.Errorf("Error tagging the OpenSearch Domain with ARN %s", aws.StringValue(domain.ARN))
				return nil, errors.WithStackTrace(err)
			}
		}

		if configObj.OpenSearchDomain.ShouldInclude(config.ResourceValue{
			Name: domain.DomainName,
			Time: firstSeenTime,
		}) {
			domainsToNuke = append(domainsToNuke, domain.DomainName)
		}
	}

	return domainsToNuke, nil
}

// getAllActiveOpenSearchDomains filters all active OpenSearch domains, which are those that have the `Created` flag true and `Deleted` flag false.
func (osd *OpenSearchDomains) getAllActiveOpenSearchDomains() ([]*opensearchservice.DomainStatus, error) {
	allDomains := []*string{}
	resp, err := osd.Client.ListDomainNames(&opensearchservice.ListDomainNamesInput{})
	if err != nil {
		logging.Logger.Errorf("Error getting all OpenSearch domains")
		return nil, errors.WithStackTrace(err)
	}
	for _, domain := range resp.DomainNames {
		allDomains = append(allDomains, domain.DomainName)
	}

	input := &opensearchservice.DescribeDomainsInput{DomainNames: allDomains}
	describedDomains, describeErr := osd.Client.DescribeDomains(input)
	if describeErr != nil {
		logging.Logger.Errorf("Error describing Domains from input %s: ", input)
		return nil, errors.WithStackTrace(describeErr)
	}

	filteredDomains := []*opensearchservice.DomainStatus{}
	for _, domain := range describedDomains.DomainStatusList {
		if aws.BoolValue(domain.Created) && aws.BoolValue(domain.Deleted) == false {
			filteredDomains = append(filteredDomains, domain)
		}
	}
	return filteredDomains, nil
}

// Tag an OpenSearch Domain identified by the given ARN when it's first seen by cloud-nuke
func (osd *OpenSearchDomains) setFirstSeenTag(domainARN *string, timestamp time.Time) error {
	logging.Logger.Debugf("Tagging the OpenSearch Domain with ARN %s with first seen timestamp", aws.StringValue(domainARN))
	firstSeenTime := util.FormatTimestampTag(timestamp)

	input := &opensearchservice.AddTagsInput{
		ARN: domainARN,
		TagList: []*opensearchservice.Tag{
			{
				Key:   aws.String(firstSeenTagKey),
				Value: aws.String(firstSeenTime),
			},
		},
	}

	_, err := osd.Client.AddTags(input)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// getFirstSeenTag gets the `cloud-nuke-first-seen` tag value for a given OpenSearch Domain
func (osd *OpenSearchDomains) getFirstSeenTag(domainARN *string) (*time.Time, error) {
	var firstSeenTime *time.Time

	input := &opensearchservice.ListTagsInput{ARN: domainARN}
	domainTags, err := osd.Client.ListTags(input)
	if err != nil {
		logging.Logger.Errorf("Error getting the tags for OpenSearch Domain with ARN %s", aws.StringValue(domainARN))
		return firstSeenTime, errors.WithStackTrace(err)
	}

	for _, tag := range domainTags.TagList {
		if util.IsFirstSeenTag(tag.Key) {
			firstSeenTime, err := util.ParseTimestampTag(tag.Value)
			if err != nil {
				logging.Logger.Errorf("Error parsing the `cloud-nuke-first-seen` tag for OpenSearch Domain with ARN %s", aws.StringValue(domainARN))
				return firstSeenTime, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return firstSeenTime, nil
}

// nukeAll nukes the given list of OpenSearch domains concurrently. Note that the opensearchservice API
// does not support bulk delete, so this routine will spawn a goroutine for each domain that needs to be nuked so that
// they can be issued concurrently.
func (osd *OpenSearchDomains) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Logger.Debugf("No OpenSearch Domains to nuke in region %s", osd.Region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on OpenSearchDomains.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many OpenSearch Domains at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyOpenSearchDomainsErr{}
	}

	logging.Logger.Debugf("Deleting OpenSearch Domains in region %s", osd.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, domainName := range identifiers {
		errChans[i] = make(chan error, 1)
		go osd.deleteAsync(wg, errChans[i], domainName)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Opensearch",
			}, map[string]interface{}{
				"region": osd.Region,
			})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	// Now wait until the OpenSearch Domains are deleted
	err := retry.DoWithRetry(
		logging.Logger,
		"Waiting for all OpenSearch Domains to be deleted.",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			resp, err := osd.Client.DescribeDomains(&opensearchservice.DescribeDomainsInput{DomainNames: identifiers})
			if err != nil {
				return errors.WithStackTrace(retry.FatalError{Underlying: err})
			}
			if len(resp.DomainStatusList) == 0 {
				return nil
			}
			return fmt.Errorf("Not all OpenSearch domains are deleted.")
		},
	)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	for _, domainName := range identifiers {
		logging.Logger.Debugf("[OK] OpenSearch Domain %s was deleted in %s", aws.StringValue(domainName), osd.Region)
	}
	return nil
}

// deleteAsync deletes the provided OpenSearch Domain asynchronously in a goroutine, using wait groups
// for concurrency control and a return channel for errors.
func (osd *OpenSearchDomains) deleteAsync(wg *sync.WaitGroup, errChan chan error, domainName *string) {
	defer wg.Done()

	input := &opensearchservice.DeleteDomainInput{DomainName: domainName}
	_, err := osd.Client.DeleteDomain(input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(domainName),
		ResourceType: "OpenSearch Domain",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}

// Custom errors

type TooManyOpenSearchDomainsErr struct{}

func (err TooManyOpenSearchDomainsErr) Error() string {
	return "Too many OpenSearch Domains requested at once."
}

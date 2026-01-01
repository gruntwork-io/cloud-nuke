package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/opensearch/types"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"
)

// OpenSearchDomainsAPI defines the interface for OpenSearch operations.
type OpenSearchDomainsAPI interface {
	AddTags(ctx context.Context, params *opensearch.AddTagsInput, optFns ...func(*opensearch.Options)) (*opensearch.AddTagsOutput, error)
	DeleteDomain(ctx context.Context, params *opensearch.DeleteDomainInput, optFns ...func(*opensearch.Options)) (*opensearch.DeleteDomainOutput, error)
	DescribeDomains(ctx context.Context, params *opensearch.DescribeDomainsInput, optFns ...func(*opensearch.Options)) (*opensearch.DescribeDomainsOutput, error)
	ListDomainNames(ctx context.Context, params *opensearch.ListDomainNamesInput, optFns ...func(*opensearch.Options)) (*opensearch.ListDomainNamesOutput, error)
	ListTags(ctx context.Context, params *opensearch.ListTagsInput, optFns ...func(*opensearch.Options)) (*opensearch.ListTagsOutput, error)
}

// NewOpenSearchDomains creates a new OpenSearchDomains resource using the generic resource pattern.
func NewOpenSearchDomains() AwsResource {
	return NewAwsResource(&resource.Resource[OpenSearchDomainsAPI]{
		ResourceTypeName: "opensearchdomain",
		BatchSize:        10,
		InitClient: func(r *resource.Resource[OpenSearchDomainsAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for OpenSearch client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = opensearch.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.OpenSearchDomain
		},
		Lister: listOpenSearchDomains,
		Nuker:  resource.ConcurrentDeleteThenWaitAll(deleteOpenSearchDomain, waitForOpenSearchDomainsDeleted),
	})
}

// listOpenSearchDomains queries AWS for all active domains in the account that meet the nuking criteria based on
// the excludeAfter and configObj configurations. Note that OpenSearch Domains do not have resource timestamps, so we
// use the first-seen tagging pattern to track which OpenSearch Domains should be nuked based on time. This routine will
// tag resources with the first-seen tag if it does not have one.
func listOpenSearchDomains(ctx context.Context, client OpenSearchDomainsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var firstSeenTime *time.Time
	var err error
	domains, err := getAllActiveOpenSearchDomains(ctx, client)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	excludeFirstSeenTag, err := util.GetBoolFromContext(ctx, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	domainsToNuke := []*string{}
	for _, domain := range domains {
		if !excludeFirstSeenTag {
			firstSeenTime, err = getOpenSearchFirstSeenTag(ctx, client, domain.ARN)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if firstSeenTime == nil {
				err := setOpenSearchFirstSeenTag(ctx, client, domain.ARN, time.Now().UTC())
				if err != nil {
					logging.Errorf("Error tagging the OpenSearch Domain with ARN %s with error: %s", aws.ToString(domain.ARN), err.Error())
					return nil, errors.WithStackTrace(err)
				}
			}
		}
		if cfg.ShouldInclude(config.ResourceValue{
			Name: domain.DomainName,
			Time: firstSeenTime,
		}) {
			domainsToNuke = append(domainsToNuke, domain.DomainName)
		}
	}

	return domainsToNuke, nil
}

// getAllActiveOpenSearchDomains filters all active OpenSearch domains, which are those that have the `Created` flag true and `Deleted` flag false.
func getAllActiveOpenSearchDomains(ctx context.Context, client OpenSearchDomainsAPI) ([]types.DomainStatus, error) {
	allDomains := []*string{}
	resp, err := client.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		logging.Errorf("Error getting all OpenSearch domains")
		return nil, errors.WithStackTrace(err)
	}
	for _, domain := range resp.DomainNames {
		allDomains = append(allDomains, domain.DomainName)
	}

	input := &opensearch.DescribeDomainsInput{DomainNames: aws.ToStringSlice(allDomains)}
	describedDomains, describeErr := client.DescribeDomains(ctx, input)
	if describeErr != nil {
		logging.Errorf("Error describing Domains from input %s: ", input)
		return nil, errors.WithStackTrace(describeErr)
	}

	filteredDomains := []types.DomainStatus{}
	for _, domain := range describedDomains.DomainStatusList {
		if aws.ToBool(domain.Created) && aws.ToBool(domain.Deleted) == false {
			filteredDomains = append(filteredDomains, domain)
		}
	}
	return filteredDomains, nil
}

// setOpenSearchFirstSeenTag tags an OpenSearch Domain identified by the given ARN when it's first seen by cloud-nuke
func setOpenSearchFirstSeenTag(ctx context.Context, client OpenSearchDomainsAPI, domainARN *string, timestamp time.Time) error {
	logging.Debugf("Tagging the OpenSearch Domain with ARN %s with first seen timestamp", aws.ToString(domainARN))
	firstSeenTime := util.FormatTimestamp(timestamp)

	input := &opensearch.AddTagsInput{
		ARN: domainARN,
		TagList: []types.Tag{
			{
				Key:   aws.String(firstSeenTagKey),
				Value: aws.String(firstSeenTime),
			},
		},
	}

	_, err := client.AddTags(ctx, input)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// getOpenSearchFirstSeenTag gets the `cloud-nuke-first-seen` tag value for a given OpenSearch Domain
func getOpenSearchFirstSeenTag(ctx context.Context, client OpenSearchDomainsAPI, domainARN *string) (*time.Time, error) {
	var firstSeenTime *time.Time

	input := &opensearch.ListTagsInput{ARN: domainARN}
	domainTags, err := client.ListTags(ctx, input)
	if err != nil {
		logging.Errorf("Error getting the tags for OpenSearch Domain with ARN %s", aws.ToString(domainARN))
		return firstSeenTime, errors.WithStackTrace(err)
	}

	for _, tag := range domainTags.TagList {
		if util.IsFirstSeenTag(tag.Key) {
			firstSeenTime, err := util.ParseTimestamp(tag.Value)
			if err != nil {
				logging.Errorf("Error parsing the `cloud-nuke-first-seen` tag for OpenSearch Domain with ARN %s", aws.ToString(domainARN))
				return firstSeenTime, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return firstSeenTime, nil
}

// deleteOpenSearchDomain deletes a single OpenSearch domain.
func deleteOpenSearchDomain(ctx context.Context, client OpenSearchDomainsAPI, domainName *string) error {
	input := &opensearch.DeleteDomainInput{DomainName: domainName}
	_, err := client.DeleteDomain(ctx, input)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

// waitForOpenSearchDomainsDeleted waits for all OpenSearch domains to be fully deleted.
func waitForOpenSearchDomainsDeleted(ctx context.Context, client OpenSearchDomainsAPI, names []string) error {
	return retry.DoWithRetry(
		logging.Logger.WithTime(time.Now()),
		"Waiting for all OpenSearch Domains to be deleted.",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			resp, err := client.DescribeDomains(ctx, &opensearch.DescribeDomainsInput{DomainNames: names})
			if err != nil {
				return errors.WithStackTrace(retry.FatalError{Underlying: err})
			}
			if len(resp.DomainStatusList) == 0 {
				return nil
			}
			return fmt.Errorf("Not all OpenSearch domains are deleted.")
		},
	)
}

// Custom errors

type TooManyOpenSearchDomainsErr struct{}

func (err TooManyOpenSearchDomainsErr) Error() string {
	return "Too many OpenSearch Domains requested at once."
}

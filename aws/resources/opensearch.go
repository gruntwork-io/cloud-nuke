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

// maxDescribeDomainsPerRequest is the AWS limit for DescribeDomains API
const maxDescribeDomainsPerRequest = 5

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
		InitClient: WrapAwsInitClient(func(r *resource.Resource[OpenSearchDomainsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = opensearch.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.OpenSearchDomain
		},
		Lister: listOpenSearchDomains,
		Nuker:  resource.ConcurrentDeleteThenWaitAll(deleteOpenSearchDomain, waitForOpenSearchDomainsDeleted),
	})
}

// listOpenSearchDomains queries AWS for all active domains that meet the nuking criteria.
// OpenSearch Domains do not have resource timestamps, so we use the first-seen tagging pattern.
func listOpenSearchDomains(ctx context.Context, client OpenSearchDomainsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	domains, err := getAllActiveOpenSearchDomains(ctx, client)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	excludeFirstSeenTag, err := util.GetBoolFromContext(ctx, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var domainsToNuke []*string
	for _, domain := range domains {
		var firstSeenTime *time.Time

		if !excludeFirstSeenTag {
			firstSeenTime, err = getFirstSeenOrSet(ctx, client, domain.ARN)
			if err != nil {
				return nil, errors.WithStackTrace(err)
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

// getFirstSeenOrSet retrieves the first-seen timestamp or sets it if not present.
func getFirstSeenOrSet(ctx context.Context, client OpenSearchDomainsAPI, domainARN *string) (*time.Time, error) {
	firstSeenTime, err := getOpenSearchFirstSeenTag(ctx, client, domainARN)
	if err != nil {
		return nil, err
	}

	if firstSeenTime == nil {
		if err := setOpenSearchFirstSeenTag(ctx, client, domainARN, time.Now().UTC()); err != nil {
			return nil, err
		}
	}

	return firstSeenTime, nil
}

// getAllActiveOpenSearchDomains filters all active OpenSearch domains, which are those that have the `Created` flag true and `Deleted` flag false.
// Note: ListDomainNames API does not support pagination but returns all domains.
// DescribeDomains API has a limit of 5 domains per request, so we batch accordingly.
func getAllActiveOpenSearchDomains(ctx context.Context, client OpenSearchDomainsAPI) ([]types.DomainStatus, error) {
	resp, err := client.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	if len(resp.DomainNames) == 0 {
		return nil, nil
	}

	// Collect all domain names
	allDomainNames := make([]string, 0, len(resp.DomainNames))
	for _, domain := range resp.DomainNames {
		allDomainNames = append(allDomainNames, aws.ToString(domain.DomainName))
	}

	// DescribeDomains API has a limit of 5 domains per request, batch accordingly
	var filteredDomains []types.DomainStatus
	for i := 0; i < len(allDomainNames); i += maxDescribeDomainsPerRequest {
		end := i + maxDescribeDomainsPerRequest
		if end > len(allDomainNames) {
			end = len(allDomainNames)
		}
		batch := allDomainNames[i:end]

		describedDomains, err := client.DescribeDomains(ctx, &opensearch.DescribeDomainsInput{
			DomainNames: batch,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, domain := range describedDomains.DomainStatusList {
			if aws.ToBool(domain.Created) && !aws.ToBool(domain.Deleted) {
				filteredDomains = append(filteredDomains, domain)
			}
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

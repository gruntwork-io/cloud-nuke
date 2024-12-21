package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type OpenSearchDomainsAPI interface {
	AddTags(ctx context.Context, params *opensearch.AddTagsInput, optFns ...func(*opensearch.Options)) (*opensearch.AddTagsOutput, error)
	DeleteDomain(ctx context.Context, params *opensearch.DeleteDomainInput, optFns ...func(*opensearch.Options)) (*opensearch.DeleteDomainOutput, error)
	DescribeDomains(ctx context.Context, params *opensearch.DescribeDomainsInput, optFns ...func(*opensearch.Options)) (*opensearch.DescribeDomainsOutput, error)
	ListDomainNames(ctx context.Context, params *opensearch.ListDomainNamesInput, optFns ...func(*opensearch.Options)) (*opensearch.ListDomainNamesOutput, error)
	ListTags(ctx context.Context, params *opensearch.ListTagsInput, optFns ...func(*opensearch.Options)) (*opensearch.ListTagsOutput, error)
}

// OpenSearchDomains represents all OpenSearch domains found in a region
type OpenSearchDomains struct {
	BaseAwsResource
	Client      OpenSearchDomainsAPI
	Region      string
	DomainNames []string
}

func (osd *OpenSearchDomains) InitV2(cfg aws.Config) {
	osd.Client = opensearch.NewFromConfig(cfg)
}

// ResourceName is the simple name of the aws resource
func (osd *OpenSearchDomains) ResourceName() string {
	return "opensearchdomain"
}

// ResourceIdentifiers the collected OpenSearch Domains
func (osd *OpenSearchDomains) ResourceIdentifiers() []string {
	return osd.DomainNames
}

// MaxBatchSize returns the number of resources that should be nuked at a time. A small number is used to ensure AWS
// doesn't throttle. OpenSearch Domains do not support bulk delete, so we will be deleting this many in parallel
// using go routines. We conservatively pick 10 here, both to limit overloading the runtime and to avoid AWS throttling
// with many API calls.
func (osd *OpenSearchDomains) MaxBatchSize() int {
	return 10
}

func (osd *OpenSearchDomains) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.OpenSearchDomain
}

func (osd *OpenSearchDomains) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := osd.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	osd.DomainNames = aws.ToStringSlice(identifiers)
	return osd.DomainNames, nil
}

// Nuke nukes all OpenSearch domain resources
func (osd *OpenSearchDomains) Nuke(identifiers []string) error {
	if err := osd.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

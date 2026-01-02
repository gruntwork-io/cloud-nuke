package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/opensearch/types"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedOpenSearch struct {
	OpenSearchDomainsAPI
	ListDomainNamesOutput opensearch.ListDomainNamesOutput
	DescribeDomainsOutput opensearch.DescribeDomainsOutput
	ListTagsOutput        opensearch.ListTagsOutput
	DeleteDomainOutput    opensearch.DeleteDomainOutput
}

func (m mockedOpenSearch) DeleteDomain(ctx context.Context, params *opensearch.DeleteDomainInput, optFns ...func(*opensearch.Options)) (*opensearch.DeleteDomainOutput, error) {
	return &m.DeleteDomainOutput, nil
}

func (m mockedOpenSearch) ListDomainNames(ctx context.Context, params *opensearch.ListDomainNamesInput, optFns ...func(*opensearch.Options)) (*opensearch.ListDomainNamesOutput, error) {
	return &m.ListDomainNamesOutput, nil
}

func (m mockedOpenSearch) DescribeDomains(ctx context.Context, params *opensearch.DescribeDomainsInput, optFns ...func(*opensearch.Options)) (*opensearch.DescribeDomainsOutput, error) {
	return &m.DescribeDomainsOutput, nil
}

func (m mockedOpenSearch) ListTags(ctx context.Context, params *opensearch.ListTagsInput, optFns ...func(*opensearch.Options)) (*opensearch.ListTagsOutput, error) {
	return &m.ListTagsOutput, nil
}

func TestOpenSearch_GetAll(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	now := time.Now()

	testName1 := "test-domain1"
	testName2 := "test-domain2"

	mockClient := mockedOpenSearch{
		ListDomainNamesOutput: opensearch.ListDomainNamesOutput{
			DomainNames: []types.DomainInfo{
				{DomainName: aws.String(testName1)},
				{DomainName: aws.String(testName2)},
			},
		},
		ListTagsOutput: opensearch.ListTagsOutput{
			TagList: []types.Tag{{
				Key:   aws.String(firstSeenTagKey),
				Value: aws.String(util.FormatTimestamp(now)),
			}},
		},
		DescribeDomainsOutput: opensearch.DescribeDomainsOutput{
			DomainStatusList: []types.DomainStatus{
				{DomainName: aws.String(testName1), Created: aws.Bool(true), Deleted: aws.Bool(false)},
				{DomainName: aws.String(testName2), Created: aws.Bool(true), Deleted: aws.Bool(false)},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listOpenSearchDomains(ctx, mockClient, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestOpenSearch_GetAll_FiltersDeletedDomains(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, true)

	mockClient := mockedOpenSearch{
		ListDomainNamesOutput: opensearch.ListDomainNamesOutput{
			DomainNames: []types.DomainInfo{
				{DomainName: aws.String("active-domain")},
				{DomainName: aws.String("deleted-domain")},
				{DomainName: aws.String("not-created-domain")},
			},
		},
		DescribeDomainsOutput: opensearch.DescribeDomainsOutput{
			DomainStatusList: []types.DomainStatus{
				{DomainName: aws.String("active-domain"), Created: aws.Bool(true), Deleted: aws.Bool(false)},
				{DomainName: aws.String("deleted-domain"), Created: aws.Bool(true), Deleted: aws.Bool(true)},
				{DomainName: aws.String("not-created-domain"), Created: aws.Bool(false), Deleted: aws.Bool(false)},
			},
		},
	}

	names, err := listOpenSearchDomains(ctx, mockClient, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"active-domain"}, aws.ToStringSlice(names))
}

func TestOpenSearch_Nuke(t *testing.T) {
	t.Parallel()

	mockClient := mockedOpenSearch{
		DeleteDomainOutput:    opensearch.DeleteDomainOutput{},
		DescribeDomainsOutput: opensearch.DescribeDomainsOutput{},
	}

	err := deleteOpenSearchDomain(context.Background(), mockClient, aws.String("test-domain"))
	require.NoError(t, err)
}

func TestOpenSearch_WaitForDeleted(t *testing.T) {
	t.Parallel()

	// Mock returns empty list, meaning domains are deleted
	mockClient := mockedOpenSearch{
		DescribeDomainsOutput: opensearch.DescribeDomainsOutput{
			DomainStatusList: []types.DomainStatus{},
		},
	}

	err := waitForOpenSearchDomainsDeleted(context.Background(), mockClient, []string{"test-domain"})
	require.NoError(t, err)
}

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

// Test we can create an OpenSearch Domain, tag it, and then find the tag
func TestOpenSearch_GetAll(t *testing.T) {

	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testName1 := "test-domain1"
	testName2 := "test-domain2"
	now := time.Now()
	osd := OpenSearchDomains{
		Client: mockedOpenSearch{
			ListDomainNamesOutput: opensearch.ListDomainNamesOutput{
				DomainNames: []types.DomainInfo{
					{DomainName: aws.String(testName1)},
					{DomainName: aws.String(testName2)},
				},
			},

			ListTagsOutput: opensearch.ListTagsOutput{
				TagList: []types.Tag{
					{
						Key:   aws.String(firstSeenTagKey),
						Value: aws.String(util.FormatTimestamp(now)),
					},
					{
						Key:   aws.String(firstSeenTagKey),
						Value: aws.String(util.FormatTimestamp(now.Add(1))),
					},
				},
			},

			DescribeDomainsOutput: opensearch.DescribeDomainsOutput{
				DomainStatusList: []types.DomainStatus{
					{
						DomainName: aws.String(testName1),
						Created:    aws.Bool(true),
						Deleted:    aws.Bool(false),
					},
					{
						DomainName: aws.String(testName2),
						Created:    aws.Bool(true),
						Deleted:    aws.Bool(false),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := osd.getAll(tc.ctx, config.Config{
				OpenSearchDomain: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestOpenSearch_NukeAll(t *testing.T) {

	t.Parallel()

	osd := OpenSearchDomains{
		Client: mockedOpenSearch{
			DeleteDomainOutput:    opensearch.DeleteDomainOutput{},
			DescribeDomainsOutput: opensearch.DescribeDomainsOutput{},
		},
	}

	err := osd.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

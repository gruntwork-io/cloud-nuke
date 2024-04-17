package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/opensearchservice"
	"github.com/aws/aws-sdk-go/service/opensearchservice/opensearchserviceiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedOpenSearch struct {
	opensearchserviceiface.OpenSearchServiceAPI
	ListDomainNamesOutput opensearchservice.ListDomainNamesOutput
	DescribeDomainsOutput opensearchservice.DescribeDomainsOutput
	ListTagsOutput        opensearchservice.ListTagsOutput
	DeleteDomainOutput    opensearchservice.DeleteDomainOutput
}

func (m mockedOpenSearch) DeleteDomain(*opensearchservice.DeleteDomainInput) (*opensearchservice.DeleteDomainOutput, error) {
	return &m.DeleteDomainOutput, nil
}

func (m mockedOpenSearch) ListDomainNames(*opensearchservice.ListDomainNamesInput) (*opensearchservice.ListDomainNamesOutput, error) {
	return &m.ListDomainNamesOutput, nil
}

func (m mockedOpenSearch) DescribeDomains(*opensearchservice.DescribeDomainsInput) (*opensearchservice.DescribeDomainsOutput, error) {
	return &m.DescribeDomainsOutput, nil
}

func (m mockedOpenSearch) ListTags(*opensearchservice.ListTagsInput) (*opensearchservice.ListTagsOutput, error) {
	return &m.ListTagsOutput, nil
}

// Test we can create an OpenSearch Domain, tag it, and then find the tag
func TestOpenSearch_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-domain1"
	testName2 := "test-domain2"
	now := time.Now()
	osd := OpenSearchDomains{
		Client: mockedOpenSearch{
			ListDomainNamesOutput: opensearchservice.ListDomainNamesOutput{
				DomainNames: []*opensearchservice.DomainInfo{
					{DomainName: aws.String(testName1)},
					{DomainName: aws.String(testName2)},
				},
			},

			ListTagsOutput: opensearchservice.ListTagsOutput{
				TagList: []*opensearchservice.Tag{
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

			DescribeDomainsOutput: opensearchservice.DescribeDomainsOutput{
				DomainStatusList: []*opensearchservice.DomainStatus{
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
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := osd.getAll(context.Background(), config.Config{
				OpenSearchDomain: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestOpenSearch_NukeAll(t *testing.T) {

	t.Parallel()

	osd := OpenSearchDomains{
		Client: mockedOpenSearch{
			DeleteDomainOutput:    opensearchservice.DeleteDomainOutput{},
			DescribeDomainsOutput: opensearchservice.DescribeDomainsOutput{},
		},
	}

	err := osd.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedRdsProxy struct {
	RdsProxyAPI
	DescribeDBProxiesOutput rds.DescribeDBProxiesOutput
	DeleteDBProxyOutput     rds.DeleteDBProxyOutput
	TagsByARN               map[string][]types.Tag
}

func (m mockedRdsProxy) DescribeDBProxies(ctx context.Context, params *rds.DescribeDBProxiesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBProxiesOutput, error) {
	return &m.DescribeDBProxiesOutput, nil
}

func (m mockedRdsProxy) DeleteDBProxy(ctx context.Context, params *rds.DeleteDBProxyInput, optFns ...func(*rds.Options)) (*rds.DeleteDBProxyOutput, error) {
	return &m.DeleteDBProxyOutput, nil
}

func (m mockedRdsProxy) ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error) {
	return &rds.ListTagsForResourceOutput{TagList: m.TagsByARN[aws.ToString(params.ResourceName)]}, nil
}

func TestRdsProxy_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	testArn1 := "arn:aws:rds:us-east-1:123456789:db-proxy:" + testName1
	testArn2 := "arn:aws:rds:us-east-1:123456789:db-proxy:" + testName2
	now := time.Now()
	client := mockedRdsProxy{
		DescribeDBProxiesOutput: rds.DescribeDBProxiesOutput{
			DBProxies: []types.DBProxy{
				{
					DBProxyName: &testName1,
					DBProxyArn:  aws.String(testArn1),
					CreatedDate: &now,
				},
				{
					DBProxyName: &testName2,
					DBProxyArn:  aws.String(testArn2),
					CreatedDate: aws.Time(now.Add(1)),
				},
			},
		},
		TagsByARN: map[string][]types.Tag{
			testArn1: {{Key: aws.String("env"), Value: aws.String("prod")}},
			testArn2: {{Key: aws.String("env"), Value: aws.String("dev")}},
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
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("^prod$")},
					},
				},
			},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listRdsProxies(
				context.Background(),
				client,
				resource.Scope{},
				tc.configObj,
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestRdsProxy_NukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-proxy"
	client := mockedRdsProxy{
		DeleteDBProxyOutput: rds.DeleteDBProxyOutput{},
	}

	err := deleteRdsProxy(context.Background(), client, &testName)
	assert.NoError(t, err)
}

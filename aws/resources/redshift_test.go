package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedRedshift struct {
	RedshiftClustersAPI

	DeleteClusterOutput    redshift.DeleteClusterOutput
	DescribeClustersOutput redshift.DescribeClustersOutput
	DescribeClusterError   error
}

func (m mockedRedshift) DescribeClusters(ctx context.Context, input *redshift.DescribeClustersInput, opts ...func(*redshift.Options)) (*redshift.DescribeClustersOutput, error) {
	return &m.DescribeClustersOutput, m.DescribeClusterError
}

func (m mockedRedshift) DeleteCluster(ctx context.Context, input *redshift.DeleteClusterInput, opts ...func(*redshift.Options)) (*redshift.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func (m mockedRedshift) WaitForOutput(ctx context.Context, params *redshift.DescribeClustersInput, maxWaitDur time.Duration, optFns ...func(*redshift.Options)) (*redshift.DescribeClustersOutput, error) {
	return nil, nil
}
func TestRedshiftCluster_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	testName1 := "test-cluster1"
	testName2 := "test-cluster2"
	rc := RedshiftClusters{
		Client: mockedRedshift{
			DescribeClustersOutput: redshift.DescribeClustersOutput{
				Clusters: []types.Cluster{
					{
						ClusterIdentifier: aws.String(testName1),
						ClusterCreateTime: aws.Time(now),
					},
					{
						ClusterIdentifier: aws.String(testName2),
						ClusterCreateTime: aws.Time(now.Add(1)),
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
			names, err := rc.getAll(context.Background(), config.Config{
				Redshift: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestRedshiftCluster_NukeAll(t *testing.T) {

	t.Parallel()

	rc := RedshiftClusters{
		Client: mockedRedshift{
			DeleteClusterOutput: redshift.DeleteClusterOutput{},
			DescribeClusterError: &smithy.GenericAPIError{
				Code: "ClusterNotFound",
			},
		},
	}
	rc.Context = context.Background()
	rc.Timeout = DefaultWaitTimeout

	err := rc.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

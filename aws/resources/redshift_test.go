package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/redshift/redshiftiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedRedshift struct {
	redshiftiface.RedshiftAPI

	DeleteClusterOutput    redshift.DeleteClusterOutput
	DescribeClustersOutput redshift.DescribeClustersOutput
}

func (m mockedRedshift) DescribeClustersPagesWithContext(_ awsgo.Context, _ *redshift.DescribeClustersInput, fn func(*redshift.DescribeClustersOutput, bool) bool, _ ...request.Option) error {
	fn(&m.DescribeClustersOutput, true)
	return nil
}

func (m mockedRedshift) DeleteClusterWithContext(_ awsgo.Context, _ *redshift.DeleteClusterInput, _ ...request.Option) (*redshift.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func (m mockedRedshift) WaitUntilClusterDeletedWithContext(_ awsgo.Context, _ *redshift.DescribeClustersInput, _ ...request.WaiterOption) error {
	return nil
}

func TestRedshiftCluster_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	testName1 := "test-cluster1"
	testName2 := "test-cluster2"
	rc := RedshiftClusters{
		Client: mockedRedshift{
			DescribeClustersOutput: redshift.DescribeClustersOutput{
				Clusters: []*redshift.Cluster{
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestRedshiftCluster_NukeAll(t *testing.T) {

	t.Parallel()

	rc := RedshiftClusters{
		Client: mockedRedshift{
			DeleteClusterOutput: redshift.DeleteClusterOutput{},
		},
	}

	err := rc.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

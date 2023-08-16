package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/redshift/redshiftiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedRedshift struct {
	redshiftiface.RedshiftAPI

	DeleteClusterOutput    redshift.DeleteClusterOutput
	DescribeClustersOutput redshift.DescribeClustersOutput
}

func (m mockedRedshift) DescribeClustersPages(input *redshift.DescribeClustersInput, fn func(*redshift.DescribeClustersOutput, bool) bool) error {
	fn(&m.DescribeClustersOutput, true)
	return nil
}

func (m mockedRedshift) DeleteCluster(input *redshift.DeleteClusterInput) (*redshift.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func (m mockedRedshift) WaitUntilClusterDeleted(*redshift.DescribeClustersInput) error {
	return nil
}

func TestRedshiftCluster_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
			names, err := rc.getAll(config.Config{
				Redshift: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestRedshiftCluster_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	rc := RedshiftClusters{
		Client: mockedRedshift{
			DeleteClusterOutput: redshift.DeleteClusterOutput{},
		},
	}

	err := rc.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

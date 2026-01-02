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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedRedshiftClient struct {
	DescribeClustersOutput redshift.DescribeClustersOutput
	DescribeClustersError  error
	DeleteClusterOutput    redshift.DeleteClusterOutput
	DeleteClusterError     error
}

func (m *mockedRedshiftClient) DescribeClusters(ctx context.Context, input *redshift.DescribeClustersInput, opts ...func(*redshift.Options)) (*redshift.DescribeClustersOutput, error) {
	return &m.DescribeClustersOutput, m.DescribeClustersError
}

func (m *mockedRedshiftClient) DeleteCluster(ctx context.Context, input *redshift.DeleteClusterInput, opts ...func(*redshift.Options)) (*redshift.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, m.DeleteClusterError
}

func TestListRedshiftClusters(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testName1 := "test-cluster-1"
	testName2 := "test-cluster-2"

	mock := &mockedRedshiftClient{
		DescribeClustersOutput: redshift.DescribeClustersOutput{
			Clusters: []types.Cluster{
				{
					ClusterIdentifier: aws.String(testName1),
					ClusterCreateTime: aws.Time(now),
				},
				{
					ClusterIdentifier: aws.String(testName2),
					ClusterCreateTime: aws.Time(now.Add(1 * time.Hour)),
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
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listRedshiftClusters(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteRedshiftCluster(t *testing.T) {
	t.Parallel()

	mock := &mockedRedshiftClient{}
	err := deleteRedshiftCluster(context.Background(), mock, aws.String("test-cluster"))
	require.NoError(t, err)
}

func TestWaitForRedshiftClusterDeleted(t *testing.T) {
	t.Parallel()

	// The waiter expects a ClusterNotFound error to indicate successful deletion.
	// When DescribeClusters returns this error, the waiter considers the cluster deleted.
	mock := &mockedRedshiftClient{
		DescribeClustersError: &smithy.GenericAPIError{
			Code: "ClusterNotFound",
		},
	}

	err := waitForRedshiftClusterDeleted(context.Background(), mock, aws.String("test-cluster"))
	require.NoError(t, err)
}

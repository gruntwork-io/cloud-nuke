package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockDBGlobalClusterMembershipsClient struct {
	DescribeGlobalClustersOutput  rds.DescribeGlobalClustersOutput
	DescribeGlobalClustersError   error
	RemoveFromGlobalClusterOutput rds.RemoveFromGlobalClusterOutput
	RemoveFromGlobalClusterError  error
	RemoveFromGlobalClusterCalls  []string // Track which clusters were removed
	removedMembers                map[string]bool
}

func (m *mockDBGlobalClusterMembershipsClient) DescribeGlobalClusters(ctx context.Context, params *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error) {
	// Filter out removed members to simulate actual AWS behavior
	if m.removedMembers != nil && len(m.DescribeGlobalClustersOutput.GlobalClusters) > 0 {
		output := m.DescribeGlobalClustersOutput
		for i := range output.GlobalClusters {
			var remainingMembers []types.GlobalClusterMember
			for _, member := range output.GlobalClusters[i].GlobalClusterMembers {
				if !m.removedMembers[aws.ToString(member.DBClusterArn)] {
					remainingMembers = append(remainingMembers, member)
				}
			}
			output.GlobalClusters[i].GlobalClusterMembers = remainingMembers
		}
		return &output, m.DescribeGlobalClustersError
	}
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func (m *mockDBGlobalClusterMembershipsClient) RemoveFromGlobalCluster(ctx context.Context, params *rds.RemoveFromGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.RemoveFromGlobalClusterOutput, error) {
	if m.RemoveFromGlobalClusterCalls == nil {
		m.RemoveFromGlobalClusterCalls = []string{}
	}
	if m.removedMembers == nil {
		m.removedMembers = make(map[string]bool)
	}
	clusterID := aws.ToString(params.DbClusterIdentifier)
	m.RemoveFromGlobalClusterCalls = append(m.RemoveFromGlobalClusterCalls, clusterID)
	// Mark as removed so DescribeGlobalClusters won't return it
	m.removedMembers[clusterID] = true
	return &m.RemoveFromGlobalClusterOutput, m.RemoveFromGlobalClusterError
}

func TestListDBGlobalClusterMemberships(t *testing.T) {
	t.Parallel()

	testName1 := "test-global-cluster1"
	testName2 := "test-global-cluster2"

	mock := &mockDBGlobalClusterMembershipsClient{
		DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
			GlobalClusters: []types.GlobalCluster{
				{GlobalClusterIdentifier: aws.String(testName1)},
				{GlobalClusterIdentifier: aws.String(testName2)},
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listDBGlobalClusterMemberships(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestRemoveGlobalClusterMemberships(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		region          string
		members         []types.GlobalClusterMember
		expectedRemoved []string
		expectError     bool
	}{
		"removes members in matching region": {
			region: "us-east-1",
			members: []types.GlobalClusterMember{
				{DBClusterArn: aws.String("arn:aws:rds:us-east-1:123456789:cluster:cluster-1")},
				{DBClusterArn: aws.String("arn:aws:rds:us-west-2:123456789:cluster:cluster-2")},
			},
			expectedRemoved: []string{"arn:aws:rds:us-east-1:123456789:cluster:cluster-1"},
			expectError:     false,
		},
		"skips members in different region": {
			region: "us-east-1",
			members: []types.GlobalClusterMember{
				{DBClusterArn: aws.String("arn:aws:rds:eu-west-1:123456789:cluster:cluster-1")},
			},
			expectedRemoved: nil,
			expectError:     false,
		},
		"no members": {
			region:          "us-east-1",
			members:         []types.GlobalClusterMember{},
			expectedRemoved: nil,
			expectError:     false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := &mockDBGlobalClusterMembershipsClient{
				DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
					GlobalClusters: []types.GlobalCluster{
						{
							GlobalClusterIdentifier: aws.String("test-global"),
							GlobalClusterMembers:    tc.members,
						},
					},
				},
			}

			err := removeGlobalClusterMemberships(context.Background(), mock, tc.region, "test-global")

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedRemoved, mock.RemoveFromGlobalClusterCalls)
			}
		})
	}
}

func TestRemoveGlobalClusterMemberships_Error(t *testing.T) {
	t.Parallel()

	mock := &mockDBGlobalClusterMembershipsClient{
		DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
			GlobalClusters: []types.GlobalCluster{
				{
					GlobalClusterIdentifier: aws.String("test-global"),
					GlobalClusterMembers: []types.GlobalClusterMember{
						{DBClusterArn: aws.String("arn:aws:rds:us-east-1:123456789:cluster:cluster-1")},
					},
				},
			},
		},
		RemoveFromGlobalClusterError: errors.New("API error"),
	}

	err := removeGlobalClusterMemberships(context.Background(), mock, "us-east-1", "test-global")
	require.Error(t, err)
	require.Contains(t, err.Error(), "API error")
}

func TestExtractRegionFromARN(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		arn      string
		expected string
	}{
		"valid ARN": {
			arn:      "arn:aws:rds:us-east-1:123456789:cluster:my-cluster",
			expected: "us-east-1",
		},
		"different region": {
			arn:      "arn:aws:rds:eu-west-1:123456789:cluster:my-cluster",
			expected: "eu-west-1",
		},
		"invalid ARN": {
			arn:      "invalid",
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := extractRegionFromARN(tc.arn)
			require.Equal(t, tc.expected, result)
		})
	}
}

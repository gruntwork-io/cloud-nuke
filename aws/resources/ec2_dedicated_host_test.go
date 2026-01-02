package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockEC2DedicatedHostsClient struct {
	DescribeHostsOutput ec2.DescribeHostsOutput
	ReleaseHostsOutput  ec2.ReleaseHostsOutput
}

func (m *mockEC2DedicatedHostsClient) DescribeHosts(ctx context.Context, params *ec2.DescribeHostsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeHostsOutput, error) {
	return &m.DescribeHostsOutput, nil
}

func (m *mockEC2DedicatedHostsClient) ReleaseHosts(ctx context.Context, params *ec2.ReleaseHostsInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseHostsOutput, error) {
	return &m.ReleaseHostsOutput, nil
}

func TestListEC2DedicatedHosts(t *testing.T) {
	t.Parallel()

	testId1 := "test-host-id-1"
	testId2 := "test-host-id-2"
	testName1 := "test-host-name-1"
	testName2 := "test-host-name-2"
	now := time.Now()

	mock := &mockEC2DedicatedHostsClient{
		DescribeHostsOutput: ec2.DescribeHostsOutput{
			Hosts: []types.Host{
				{
					HostId: aws.String(testId1),
					Tags: []types.Tag{
						{
							Key:   aws.String("Name"),
							Value: aws.String(testName1),
						},
					},
					AllocationTime: aws.Time(now),
				},
				{
					HostId: aws.String(testId2),
					Tags: []types.Tag{
						{
							Key:   aws.String("Name"),
							Value: aws.String(testName2),
						},
					},
					AllocationTime: aws.Time(now.Add(1)),
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
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testId1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listEC2DedicatedHosts(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestReleaseEC2DedicatedHosts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		output          ec2.ReleaseHostsOutput
		hostIds         []string
		expectedSuccess int
		expectedFailure int
	}{
		"allSuccessful": {
			output: ec2.ReleaseHostsOutput{
				Successful: []string{"test-host-id-1", "test-host-id-2"},
			},
			hostIds:         []string{"test-host-id-1", "test-host-id-2"},
			expectedSuccess: 2,
			expectedFailure: 0,
		},
		"allFailed": {
			output: ec2.ReleaseHostsOutput{
				Unsuccessful: []types.UnsuccessfulItem{
					{ResourceId: aws.String("test-host-id-1"), Error: &types.UnsuccessfulItemError{Message: aws.String("error1")}},
					{ResourceId: aws.String("test-host-id-2"), Error: &types.UnsuccessfulItemError{Message: aws.String("error2")}},
				},
			},
			hostIds:         []string{"test-host-id-1", "test-host-id-2"},
			expectedSuccess: 0,
			expectedFailure: 2,
		},
		"mixed": {
			output: ec2.ReleaseHostsOutput{
				Successful: []string{"test-host-id-1"},
				Unsuccessful: []types.UnsuccessfulItem{
					{ResourceId: aws.String("test-host-id-2"), Error: &types.UnsuccessfulItemError{Message: aws.String("error")}},
				},
			},
			hostIds:         []string{"test-host-id-1", "test-host-id-2"},
			expectedSuccess: 1,
			expectedFailure: 1,
		},
		"emptyInput": {
			output:          ec2.ReleaseHostsOutput{},
			hostIds:         []string{},
			expectedSuccess: 0,
			expectedFailure: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockEC2DedicatedHostsClient{
				ReleaseHostsOutput: tc.output,
			}

			results := releaseEC2DedicatedHosts(context.Background(), mock, tc.hostIds)

			successCount := 0
			failureCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				} else {
					failureCount++
				}
			}

			require.Equal(t, tc.expectedSuccess, successCount, "success count mismatch")
			require.Equal(t, tc.expectedFailure, failureCount, "failure count mismatch")
		})
	}
}

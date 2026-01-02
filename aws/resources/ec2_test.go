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

type mockedEC2Instances struct {
	EC2InstancesAPI
	DescribeInstancesOutput         ec2.DescribeInstancesOutput
	DescribeInstanceAttributeOutput map[string]ec2.DescribeInstanceAttributeOutput
	TerminateInstancesOutput        ec2.TerminateInstancesOutput
	DescribeAddressesOutput         ec2.DescribeAddressesOutput
	ReleaseAddressOutput            ec2.ReleaseAddressOutput
}

func (m mockedEC2Instances) DescribeInstances(_ context.Context, _ *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &m.DescribeInstancesOutput, nil
}

func (m mockedEC2Instances) DescribeInstanceAttribute(_ context.Context, params *ec2.DescribeInstanceAttributeInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstanceAttributeOutput, error) {
	id := params.InstanceId
	output := m.DescribeInstanceAttributeOutput[*id]

	return &output, nil
}

func (m mockedEC2Instances) TerminateInstances(_ context.Context, _ *ec2.TerminateInstancesInput, _ ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
}

func (m mockedEC2Instances) DescribeAddresses(_ context.Context, _ *ec2.DescribeAddressesInput, _ ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}

func (m mockedEC2Instances) ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func TestEC2Instances_ResourceName(t *testing.T) {
	r := NewEC2Instances()
	require.Equal(t, "ec2", r.ResourceName())
}

func TestEC2Instances_MaxBatchSize(t *testing.T) {
	r := NewEC2Instances()
	require.Equal(t, 49, r.MaxBatchSize())
}

func TestListEC2Instances(t *testing.T) {
	t.Parallel()
	testId1 := "testId1"
	testId2 := "testId2"
	testName1 := "testName1"
	testName2 := "testName2"
	now := time.Now()

	mock := mockedEC2Instances{
		DescribeInstancesOutput: ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{
				{
					Instances: []types.Instance{
						{
							InstanceId: aws.String(testId1),
							Tags: []types.Tag{
								{
									Key:   aws.String("Name"),
									Value: aws.String(testName1),
								},
							},
							LaunchTime: aws.Time(now),
						},
						{
							InstanceId: aws.String(testId2),
							Tags: []types.Tag{
								{
									Key:   aws.String("Name"),
									Value: aws.String(testName2),
								},
							},
							LaunchTime: aws.Time(now.Add(1)),
						},
					},
				},
			},
		},
		DescribeInstanceAttributeOutput: map[string]ec2.DescribeInstanceAttributeOutput{
			testId1: {
				DisableApiTermination: &types.AttributeBooleanValue{
					Value: aws.Bool(false),
				},
			},
			testId2: {
				DisableApiTermination: &types.AttributeBooleanValue{
					Value: aws.Bool(false),
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
			names, err := listEC2Instances(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestReleaseInstanceEIPs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		describeAddressesOutput ec2.DescribeAddressesOutput
	}{
		"noEIPs": {
			describeAddressesOutput: ec2.DescribeAddressesOutput{},
		},
		"withEIP": {
			describeAddressesOutput: ec2.DescribeAddressesOutput{
				Addresses: []types.Address{
					{
						AllocationId: aws.String("alloc-test-id1"),
						InstanceId:   aws.String("testId1"),
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := mockedEC2Instances{
				DescribeAddressesOutput: tc.describeAddressesOutput,
			}
			err := releaseInstanceEIPs(context.Background(), mock, aws.String("testId1"))
			require.NoError(t, err)
		})
	}
}

func TestTerminateEC2Instance(t *testing.T) {
	t.Parallel()

	mock := mockedEC2Instances{
		TerminateInstancesOutput: ec2.TerminateInstancesOutput{},
	}

	err := terminateEC2Instance(context.Background(), mock, aws.String("testId1"))
	require.NoError(t, err)
}

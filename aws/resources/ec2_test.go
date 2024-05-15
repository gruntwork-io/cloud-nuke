package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/stretchr/testify/require"
)

type mockedEC2Instances struct {
	ec2iface.EC2API
	DescribeInstancesOutput         ec2.DescribeInstancesOutput
	DescribeInstanceAttributeOutput map[string]ec2.DescribeInstanceAttributeOutput
	TerminateInstancesOutput        ec2.TerminateInstancesOutput
	DescribeAddressesOutput         ec2.DescribeAddressesOutput
	ReleaseAddressOutput            ec2.ReleaseAddressOutput
}

func (m mockedEC2Instances) DescribeInstancesWithContext(
	_ awsgo.Context, _ *ec2.DescribeInstancesInput, _ ...request.Option) (*ec2.DescribeInstancesOutput, error) {
	return &m.DescribeInstancesOutput, nil
}

func (m mockedEC2Instances) DescribeInstanceAttributeWithContext(
	_ awsgo.Context, input *ec2.DescribeInstanceAttributeInput, _ ...request.Option) (*ec2.DescribeInstanceAttributeOutput, error) {
	id := input.InstanceId
	output := m.DescribeInstanceAttributeOutput[*id]

	return &output, nil
}

func (m mockedEC2Instances) TerminateInstancesWithContext(
	_ awsgo.Context, _ *ec2.TerminateInstancesInput, _ ...request.Option) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
}

func (m mockedEC2Instances) WaitUntilInstanceTerminatedWithContext(
	_ awsgo.Context, _ *ec2.DescribeInstancesInput, _ ...request.WaiterOption) error {
	return nil
}
func (m mockedEC2Instances) DescribeAddressesWithContext(_ awsgo.Context, _ *ec2.DescribeAddressesInput, _ ...request.Option) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}
func (m mockedEC2Instances) ReleaseAddressWithContext(_ awsgo.Context, _ *ec2.ReleaseAddressInput, _ ...request.Option) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func TestEc2Instances_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "testId1"
	testId2 := "testId2"
	testName1 := "testName1"
	testName2 := "testName2"
	now := time.Now()
	ei := EC2Instances{
		Client: mockedEC2Instances{
			DescribeInstancesOutput: ec2.DescribeInstancesOutput{
				Reservations: []*ec2.Reservation{
					{
						Instances: []*ec2.Instance{
							{
								InstanceId: awsgo.String(testId1),
								Tags: []*ec2.Tag{
									{
										Key:   awsgo.String("Name"),
										Value: awsgo.String(testName1),
									},
								},
								LaunchTime: awsgo.Time(now),
							},
							{
								InstanceId: awsgo.String(testId2),
								Tags: []*ec2.Tag{
									{
										Key:   awsgo.String("Name"),
										Value: awsgo.String(testName2),
									},
								},
								LaunchTime: awsgo.Time(now.Add(1)),
							},
						},
					},
				},
			},
			DescribeInstanceAttributeOutput: map[string]ec2.DescribeInstanceAttributeOutput{
				testId1: {
					DisableApiTermination: &ec2.AttributeBooleanValue{
						Value: awsgo.Bool(false),
					},
				},
				testId2: {
					DisableApiTermination: &ec2.AttributeBooleanValue{
						Value: awsgo.Bool(false),
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
			names, err := ei.getAll(context.Background(), config.Config{
				EC2: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestEc2Instances_NukeAll(t *testing.T) {

	t.Parallel()

	ei := EC2Instances{
		Client: mockedEC2Instances{
			TerminateInstancesOutput: ec2.TerminateInstancesOutput{},
		},
	}

	err := ei.nukeAll([]*string{awsgo.String("testId1")})
	require.NoError(t, err)
}

func TestEc2InstancesWithEIP_NukeAll(t *testing.T) {
	logging.ParseLogLevel("debug")
	t.Parallel()

	ei := EC2Instances{
		Client: mockedEC2Instances{
			TerminateInstancesOutput: ec2.TerminateInstancesOutput{},
			DescribeAddressesOutput: ec2.DescribeAddressesOutput{
				Addresses: []*ec2.Address{
					{
						AllocationId: awsgo.String("alloc-test-id1"),
						InstanceId:   awsgo.String("testId1"),
					},
				},
			},
		},
	}

	err := ei.nukeAll([]*string{awsgo.String("testId1")})
	require.NoError(t, err)
}

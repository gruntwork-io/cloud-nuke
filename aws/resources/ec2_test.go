package resources

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedEC2Instances struct {
	ec2iface.EC2API
	DescribeInstancesOutput         ec2.DescribeInstancesOutput
	DescribeInstanceAttributeOutput map[string]ec2.DescribeInstanceAttributeOutput
	TerminateInstancesOutput        ec2.TerminateInstancesOutput
}

func (m mockedEC2Instances) DescribeInstances(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return &m.DescribeInstancesOutput, nil
}

func (m mockedEC2Instances) DescribeInstanceAttribute(input *ec2.DescribeInstanceAttributeInput) (*ec2.DescribeInstanceAttributeOutput, error) {
	id := input.InstanceId
	output := m.DescribeInstanceAttributeOutput[*id]

	return &output, nil
}

func (m mockedEC2Instances) TerminateInstances(*ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
}

func (m mockedEC2Instances) WaitUntilInstanceTerminated(*ec2.DescribeInstancesInput) error {
	return nil
}

func TestEc2Instances_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	ei := EC2Instances{
		Client: mockedEC2Instances{
			TerminateInstancesOutput: ec2.TerminateInstancesOutput{},
		},
	}

	err := ei.nukeAll([]*string{awsgo.String("testId1")})
	require.NoError(t, err)
}

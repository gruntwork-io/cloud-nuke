package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedNetworkInterface struct {
	BaseAwsResource
	ec2iface.EC2API
	DescribeNetworkInterfacesOutput ec2.DescribeNetworkInterfacesOutput
	DeleteNetworkInterfaceOutput    ec2.DeleteNetworkInterfaceOutput
	DescribeAddressesOutput         ec2.DescribeAddressesOutput
	TerminateInstancesOutput        ec2.TerminateInstancesOutput
	ReleaseAddressOutput            ec2.ReleaseAddressOutput
}

func (m mockedNetworkInterface) DescribeNetworkInterfaces(*ec2.DescribeNetworkInterfacesInput) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return &m.DescribeNetworkInterfacesOutput, nil
}

func (m mockedNetworkInterface) DeleteNetworkInterface(*ec2.DeleteNetworkInterfaceInput) (*ec2.DeleteNetworkInterfaceOutput, error) {
	return &m.DeleteNetworkInterfaceOutput, nil
}

func (m mockedNetworkInterface) DescribeAddresses(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}

func (m mockedNetworkInterface) ReleaseAddress(*ec2.ReleaseAddressInput) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func (m mockedNetworkInterface) TerminateInstances(*ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
}

func (m mockedNetworkInterface) WaitUntilInstanceTerminated(*ec2.DescribeInstancesInput) error {
	return nil
}

func TestNetworkInterface_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")

	var (
		now     = time.Now()
		testId1 = "eni-09e36c45cbdbfb001"
		testId2 = "eni-09e36c45cbdbfb002"

		testName1 = "cloud-nuke-eni-001"
		testName2 = "cloud-nuke-eni-002"
	)

	resourceObject := NetworkInterface{
		Client: mockedNetworkInterface{
			DescribeNetworkInterfacesOutput: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{
					{
						NetworkInterfaceId: awsgo.String(testId1),
						TagSet: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						NetworkInterfaceId: awsgo.String(testId2),
						TagSet: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now.Add(1 * time.Hour))),
							},
						},
					},
				},
			},
		},
	}
	resourceObject.BaseAwsResource.Init(nil)

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
		"nameInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: awsgo.Time(now),
				}},
			expected: []string{
				testId1,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := resourceObject.getAll(context.Background(), config.Config{
				NetworkInterface: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestNetworkInterface_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	var (
		testId1 = "eni-09e36c45cbdbfb001"
		testId2 = "eni-09e36c45cbdbfb002"

		testName1 = "cloud-nuke-eni-001"
		testName2 = "cloud-nuke-eni-002"
	)

	resourceObject := NetworkInterface{
		BaseAwsResource: BaseAwsResource{
			Nukables: map[string]error{
				testId1: nil,
				testId2: nil,
			},
		},
		Client: mockedNetworkInterface{
			DeleteNetworkInterfaceOutput: ec2.DeleteNetworkInterfaceOutput{},
			DescribeNetworkInterfacesOutput: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{
					{
						NetworkInterfaceId: awsgo.String(testId1),
						TagSet: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
						},
						Attachment: &ec2.NetworkInterfaceAttachment{
							AttachmentId: awsgo.String("network-attachment-09e36c45cbdbfb001"),
							InstanceId:   awsgo.String("ec2-instance-09e36c45cbdbfb001"),
						},
					},
					{
						NetworkInterfaceId: awsgo.String(testId2),
						TagSet: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
						},
						Attachment: &ec2.NetworkInterfaceAttachment{
							AttachmentId: awsgo.String("network-attachment-09e36c45cbdbfb002"),
							InstanceId:   awsgo.String("ec2-instance-09e36c45cbdbfb002"),
						},
					},
				},
			},
			DescribeAddressesOutput: ec2.DescribeAddressesOutput{
				Addresses: []*ec2.Address{
					{
						AllocationId: awsgo.String("ec2-addr-alloc-09e36c45cbdbfb001"),
						InstanceId:   awsgo.String("ec2-instance-09e36c45cbdbfb001"),
					},
					{
						AllocationId: awsgo.String("ec2-addr-alloc-09e36c45cbdbfb002"),
						InstanceId:   awsgo.String("ec2-instance-09e36c45cbdbfb002"),
					},
				},
			},
			TerminateInstancesOutput: ec2.TerminateInstancesOutput{},
			ReleaseAddressOutput:     ec2.ReleaseAddressOutput{},
		},
	}

	err := resourceObject.nukeAll([]*string{
		awsgo.String(testId1),
		awsgo.String(testId2),
	})
	require.NoError(t, err)
}

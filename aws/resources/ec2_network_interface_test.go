package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
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

func (m mockedNetworkInterface) DescribeNetworkInterfacesWithContext(_ awsgo.Context, _ *ec2.DescribeNetworkInterfacesInput, _ ...request.Option) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return &m.DescribeNetworkInterfacesOutput, nil
}

func (m mockedNetworkInterface) DeleteNetworkInterface(_ *ec2.DeleteNetworkInterfaceInput) (*ec2.DeleteNetworkInterfaceOutput, error) {
	return &m.DeleteNetworkInterfaceOutput, nil
}
func (m mockedNetworkInterface) DeleteNetworkInterfaceWithContext(_ awsgo.Context, _ *ec2.DeleteNetworkInterfaceInput, _ ...request.Option) (*ec2.DeleteNetworkInterfaceOutput, error) {
	return &m.DeleteNetworkInterfaceOutput, nil
}

func (m mockedNetworkInterface) DescribeAddressesWithContext(_ awsgo.Context, _ *ec2.DescribeAddressesInput, _ ...request.Option) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}

func (m mockedNetworkInterface) ReleaseAddressWithContext(_ awsgo.Context, _ *ec2.ReleaseAddressInput, _ ...request.Option) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func (m mockedNetworkInterface) TerminateInstancesWithContext(_ awsgo.Context, _ *ec2.TerminateInstancesInput, _ ...request.Option) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
}

func (m mockedNetworkInterface) WaitUntilInstanceTerminatedWithContext(_ awsgo.Context, _ *ec2.DescribeInstancesInput, _ ...request.WaiterOption) error {
	return nil
}

func TestNetworkInterface_GetAll(t *testing.T) {

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

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
						InterfaceType:      awsgo.String(NetworkInterfaceTypeInterface),
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
						InterfaceType:      awsgo.String(NetworkInterfaceTypeInterface),
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
		ctx       context.Context
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"nameInclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId1},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
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
			names, err := resourceObject.getAll(tc.ctx, config.Config{
				NetworkInterface: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestNetworkInterface_NukeAll(t *testing.T) {

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
						InterfaceType:      awsgo.String(NetworkInterfaceTypeInterface),
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
						InterfaceType:      awsgo.String(NetworkInterfaceTypeInterface),
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

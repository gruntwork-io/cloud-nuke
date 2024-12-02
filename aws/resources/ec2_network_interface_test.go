package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedNetworkInterface struct {
	BaseAwsResource
	NetworkInterfaceAPI
	DescribeNetworkInterfacesOutput ec2.DescribeNetworkInterfacesOutput
	DeleteNetworkInterfaceOutput    ec2.DeleteNetworkInterfaceOutput
	DescribeAddressesOutput         ec2.DescribeAddressesOutput
	TerminateInstancesOutput        ec2.TerminateInstancesOutput
	ReleaseAddressOutput            ec2.ReleaseAddressOutput
	DescribeNetworkInterfacesError  error
}

func (m mockedNetworkInterface) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return &m.DescribeNetworkInterfacesOutput, m.DescribeNetworkInterfacesError
}

func (m mockedNetworkInterface) DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error) {
	return &m.DeleteNetworkInterfaceOutput, nil
}

func (m mockedNetworkInterface) DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}

func (m mockedNetworkInterface) ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func (m mockedNetworkInterface) TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
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
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: awsgo.String(testId1),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
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
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
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
			require.Equal(t, tc.expected, awsgo.ToStringSlice(names))
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
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: awsgo.String(testId1),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
						},
						Attachment: &types.NetworkInterfaceAttachment{
							AttachmentId: awsgo.String("network-attachment-09e36c45cbdbfb001"),
							InstanceId:   awsgo.String("ec2-instance-09e36c45cbdbfb001"),
						},
					},
					{
						NetworkInterfaceId: awsgo.String(testId2),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
						},
						Attachment: &types.NetworkInterfaceAttachment{
							AttachmentId: awsgo.String("network-attachment-09e36c45cbdbfb002"),
							InstanceId:   awsgo.String("ec2-instance-09e36c45cbdbfb002"),
						},
					},
				},
			},
			DescribeAddressesOutput: ec2.DescribeAddressesOutput{
				Addresses: []types.Address{
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
			DescribeNetworkInterfacesError: &smithy.GenericAPIError{
				Code: "terminated",
			},
		},
	}
	resourceObject.Context = context.Background()

	err := resourceObject.nukeAll([]*string{
		awsgo.String(testId1),
		awsgo.String(testId2),
	})
	require.NoError(t, err)
}

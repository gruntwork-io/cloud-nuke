package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	DescribeNetworkInterfacesPages  []ec2.DescribeNetworkInterfacesOutput
	DeleteNetworkInterfaceOutput    ec2.DeleteNetworkInterfaceOutput
	DescribeAddressesOutput         ec2.DescribeAddressesOutput
	TerminateInstancesOutput        ec2.TerminateInstancesOutput
	ReleaseAddressOutput            ec2.ReleaseAddressOutput
	DescribeNetworkInterfacesError  error
	pageIndex                       int
}

func (m *mockedNetworkInterface) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	if len(m.DescribeNetworkInterfacesPages) > 0 {
		if m.pageIndex >= len(m.DescribeNetworkInterfacesPages) {
			return &ec2.DescribeNetworkInterfacesOutput{}, nil
		}
		output := m.DescribeNetworkInterfacesPages[m.pageIndex]
		m.pageIndex++
		return &output, m.DescribeNetworkInterfacesError
	}
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

func (m mockedNetworkInterface) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{}, nil
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
		Client: &mockedNetworkInterface{
			DescribeNetworkInterfacesOutput: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: aws.String(testId1),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							},
							{
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						NetworkInterfaceId: aws.String(testId2),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
							{
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour))),
							},
						},
					},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx          context.Context
		configObj    config.ResourceType
		expected     []string
		resourceObj  NetworkInterface
		usePaginated bool
	}{
		"emptyFilter": {
			ctx:        ctx,
			configObj:  config.ResourceType{},
			expected:   []string{testId1, testId2},
			resourceObj: resourceObject,
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected:    []string{testId2},
			resourceObj: resourceObject,
		},
		"nameInclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected:    []string{testId1},
			resourceObj: resourceObject,
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected:    []string{testId1},
			resourceObj: resourceObject,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := tc.resourceObj.getAll(tc.ctx, config.Config{
				NetworkInterface: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestNetworkInterface_GetAll_Pagination(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	now := time.Now()

	nextToken := "next-token"

	client := &mockedNetworkInterface{
		DescribeNetworkInterfacesPages: []ec2.DescribeNetworkInterfacesOutput{
			{
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: aws.String("eni-page1-001"),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
							{Key: aws.String("Name"), Value: aws.String("eni-page1")},
							{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
						},
					},
				},
				NextToken: &nextToken,
			},
			{
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: aws.String("eni-page2-001"),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
							{Key: aws.String("Name"), Value: aws.String("eni-page2")},
							{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
						},
					},
				},
				NextToken: nil,
			},
		},
	}

	resourceObject := NetworkInterface{Client: client}
	resourceObject.Context = ctx

	identifiers, err := resourceObject.getAll(ctx, config.Config{
		NetworkInterface: config.ResourceType{},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"eni-page1-001", "eni-page2-001"}, aws.ToStringSlice(identifiers))
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
		Client: &mockedNetworkInterface{
			DeleteNetworkInterfaceOutput: ec2.DeleteNetworkInterfaceOutput{},
			DescribeNetworkInterfacesOutput: ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: aws.String(testId1),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							},
						},
						Attachment: &types.NetworkInterfaceAttachment{
							AttachmentId: aws.String("network-attachment-09e36c45cbdbfb001"),
							InstanceId:   aws.String("ec2-instance-09e36c45cbdbfb001"),
						},
					},
					{
						NetworkInterfaceId: aws.String(testId2),
						InterfaceType:      NetworkInterfaceTypeInterface,
						TagSet: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
						},
						Attachment: &types.NetworkInterfaceAttachment{
							AttachmentId: aws.String("network-attachment-09e36c45cbdbfb002"),
							InstanceId:   aws.String("ec2-instance-09e36c45cbdbfb002"),
						},
					},
				},
			},
			DescribeAddressesOutput: ec2.DescribeAddressesOutput{
				Addresses: []types.Address{
					{
						AllocationId: aws.String("ec2-addr-alloc-09e36c45cbdbfb001"),
						InstanceId:   aws.String("ec2-instance-09e36c45cbdbfb001"),
					},
					{
						AllocationId: aws.String("ec2-addr-alloc-09e36c45cbdbfb002"),
						InstanceId:   aws.String("ec2-instance-09e36c45cbdbfb002"),
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
		aws.String(testId1),
		aws.String(testId2),
	})
	require.NoError(t, err)
}

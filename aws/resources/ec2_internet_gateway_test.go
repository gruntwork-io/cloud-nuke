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
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedInternetGateway struct {
	InternetGatewayAPI
	DescribeInternetGatewaysOutput ec2.DescribeInternetGatewaysOutput
	DescribeVpcsOutput             ec2.DescribeVpcsOutput
	DetachInternetGatewayOutput    ec2.DetachInternetGatewayOutput
	DeleteInternetGatewayOutput    ec2.DeleteInternetGatewayOutput
}

func (m mockedInternetGateway) DescribeInternetGateways(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error) {
	return &m.DescribeInternetGatewaysOutput, nil
}

func (m mockedInternetGateway) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m mockedInternetGateway) DetachInternetGateway(ctx context.Context, params *ec2.DetachInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DetachInternetGatewayOutput, error) {
	return &m.DetachInternetGatewayOutput, nil
}

func (m mockedInternetGateway) DeleteInternetGateway(ctx context.Context, params *ec2.DeleteInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteInternetGatewayOutput, error) {
	return &m.DeleteInternetGatewayOutput, nil
}

func (m mockedInternetGateway) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &ec2.CreateTagsOutput{}, nil
}

func TestInternetGateway_GetAll(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	now := time.Now()
	gateway1, gateway2 := "igw-001", "igw-002"
	testName1, testName2 := "cloud-nuke-igw-001", "cloud-nuke-igw-002"

	mockClient := mockedInternetGateway{
		DescribeInternetGatewaysOutput: ec2.DescribeInternetGatewaysOutput{
			InternetGateways: []types.InternetGateway{
				{
					InternetGatewayId: aws.String(gateway1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1)))},
					},
				},
				{
					InternetGatewayId: aws.String(gateway2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1)))},
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
			expected:  []string{gateway1, gateway2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{gateway2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listInternetGateways(ctx, mockClient, resource.Scope{Region: "us-east-1"}, tc.configObj, false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestInternetGateway_NukeAll(t *testing.T) {
	t.Parallel()

	gateway1, gateway2 := "igw-001", "igw-002"
	vpcID := "vpc-test"

	// Mock client returns VPC attachment for each gateway when described
	mockClient := mockedInternetGateway{
		DescribeInternetGatewaysOutput: ec2.DescribeInternetGatewaysOutput{
			InternetGateways: []types.InternetGateway{
				{
					InternetGatewayId: aws.String(gateway1),
					Attachments: []types.InternetGatewayAttachment{
						{VpcId: aws.String(vpcID), State: types.AttachmentStatusAttached},
					},
				},
			},
		},
		DetachInternetGatewayOutput: ec2.DetachInternetGatewayOutput{},
		DeleteInternetGatewayOutput: ec2.DeleteInternetGatewayOutput{},
	}

	nuker := resource.MultiStepDeleter(detachInternetGateway, deleteInternetGateway)
	results := nuker(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, "internet-gateway", []*string{aws.String(gateway1), aws.String(gateway2)})

	require.Len(t, results, 2)
	require.Equal(t, gateway1, results[0].Identifier)
	require.NoError(t, results[0].Error)
	require.Equal(t, gateway2, results[1].Identifier)
	require.NoError(t, results[1].Error)
}

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

type mockEC2VpcClient struct {
	DescribeVpcsOutput ec2.DescribeVpcsOutput
	DeleteVpcOutput    ec2.DeleteVpcOutput
	DeleteVpcError     error
}

func (m *mockEC2VpcClient) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m *mockEC2VpcClient) DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error) {
	return &m.DeleteVpcOutput, m.DeleteVpcError
}

func (m *mockEC2VpcClient) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &ec2.CreateTagsOutput{}, nil
}

func TestListVPCs(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testId1 := "vpc-001"
	testId2 := "vpc-002"
	testName1 := "test-vpc-1"
	testName2 := "test-vpc-2"

	mock := &mockEC2VpcClient{
		DescribeVpcsOutput: ec2.DescribeVpcsOutput{
			Vpcs: []types.Vpc{
				{
					VpcId: aws.String(testId1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcId: aws.String(testId2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
		},
	}

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	tests := map[string]struct {
		cfg      config.ResourceType
		expected []string
	}{
		"emptyFilter": {
			cfg:      config.ResourceType{},
			expected: []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listVPCs(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.cfg, false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteVPC(t *testing.T) {
	t.Parallel()

	mock := &mockEC2VpcClient{}
	err := deleteVPC(context.Background(), mock, aws.String("vpc-test"))
	require.NoError(t, err)
}

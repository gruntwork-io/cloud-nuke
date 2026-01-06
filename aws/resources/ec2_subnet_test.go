package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockEC2SubnetClient struct {
	DescribeOutput ec2.DescribeSubnetsOutput
	DeleteOutput   ec2.DeleteSubnetOutput
}

func (m *mockEC2SubnetClient) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return &m.DescribeOutput, nil
}

func (m *mockEC2SubnetClient) DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error) {
	return &m.DeleteOutput, nil
}

func TestListEC2Subnets(t *testing.T) {
	t.Parallel()

	mock := &mockEC2SubnetClient{
		DescribeOutput: ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{
					SubnetId: aws.String("subnet-001"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("subnet1")},
					},
				},
				{
					SubnetId: aws.String("subnet-002"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("subnet2")},
					},
				},
			},
		},
	}

	ids, err := listEC2Subnets(context.Background(), mock, resource.Scope{}, config.ResourceType{}, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"subnet-001", "subnet-002"}, aws.ToStringSlice(ids))
}

func TestListEC2Subnets_WithFilter(t *testing.T) {
	t.Parallel()

	mock := &mockEC2SubnetClient{
		DescribeOutput: ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{
					SubnetId: aws.String("subnet-001"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("keep-this")},
					},
				},
				{
					SubnetId: aws.String("subnet-002"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("skip-this")},
					},
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	ids, err := listEC2Subnets(context.Background(), mock, resource.Scope{}, cfg, false)
	require.NoError(t, err)
	require.Equal(t, []string{"subnet-001"}, aws.ToStringSlice(ids))
}

func TestDeleteSubnet(t *testing.T) {
	t.Parallel()

	mock := &mockEC2SubnetClient{}
	err := deleteSubnet(context.Background(), mock, aws.String("subnet-test"))
	require.NoError(t, err)
}

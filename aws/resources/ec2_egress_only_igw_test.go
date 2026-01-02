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

type mockEgressOnlyIGClient struct {
	DescribeOutput ec2.DescribeEgressOnlyInternetGatewaysOutput
	DeleteOutput   ec2.DeleteEgressOnlyInternetGatewayOutput
}

func (m *mockEgressOnlyIGClient) DescribeEgressOnlyInternetGateways(ctx context.Context, params *ec2.DescribeEgressOnlyInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeEgressOnlyInternetGatewaysOutput, error) {
	return &m.DescribeOutput, nil
}

func (m *mockEgressOnlyIGClient) DeleteEgressOnlyInternetGateway(ctx context.Context, params *ec2.DeleteEgressOnlyInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteEgressOnlyInternetGatewayOutput, error) {
	return &m.DeleteOutput, nil
}

func TestListEgressOnlyInternetGateways(t *testing.T) {
	t.Parallel()

	mock := &mockEgressOnlyIGClient{
		DescribeOutput: ec2.DescribeEgressOnlyInternetGatewaysOutput{
			EgressOnlyInternetGateways: []types.EgressOnlyInternetGateway{
				{
					EgressOnlyInternetGatewayId: aws.String("eigw-001"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("gateway1")},
					},
				},
				{
					EgressOnlyInternetGatewayId: aws.String("eigw-002"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("gateway2")},
					},
				},
			},
		},
	}

	ids, err := listEgressOnlyInternetGateways(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"eigw-001", "eigw-002"}, aws.ToStringSlice(ids))
}

func TestListEgressOnlyInternetGateways_WithFilter(t *testing.T) {
	t.Parallel()

	mock := &mockEgressOnlyIGClient{
		DescribeOutput: ec2.DescribeEgressOnlyInternetGatewaysOutput{
			EgressOnlyInternetGateways: []types.EgressOnlyInternetGateway{
				{
					EgressOnlyInternetGatewayId: aws.String("eigw-001"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("keep-this")},
					},
				},
				{
					EgressOnlyInternetGatewayId: aws.String("eigw-002"),
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

	ids, err := listEgressOnlyInternetGateways(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"eigw-001"}, aws.ToStringSlice(ids))
}

func TestDeleteEgressOnlyInternetGateway(t *testing.T) {
	t.Parallel()

	mock := &mockEgressOnlyIGClient{}
	err := deleteEgressOnlyInternetGateway(context.Background(), mock, aws.String("eigw-test"))
	require.NoError(t, err)
}

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEC2EndpointsClient struct {
	DescribeVpcEndpointsOutput ec2.DescribeVpcEndpointsOutput
	DeleteVpcEndpointsOutput   ec2.DeleteVpcEndpointsOutput
}

func (m *mockEC2EndpointsClient) DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	return &m.DescribeVpcEndpointsOutput, nil
}

func (m *mockEC2EndpointsClient) DeleteVpcEndpoints(ctx context.Context, params *ec2.DeleteVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointsOutput, error) {
	return &m.DeleteVpcEndpointsOutput, nil
}

func TestEC2Endpoints_ResourceName(t *testing.T) {
	r := NewEC2Endpoints()
	assert.Equal(t, "ec2-endpoint", r.ResourceName())
}

func TestEC2Endpoints_MaxBatchSize(t *testing.T) {
	r := NewEC2Endpoints()
	assert.Equal(t, 49, r.MaxBatchSize())
}

func TestListEC2Endpoints(t *testing.T) {
	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	now := time.Now()
	endpoint1 := "vpce-0b201b2dcd4f77a2f001"
	endpoint2 := "vpce-0b201b2dcd4f77a2f002"
	testName1 := "cloud-nuke-endpoint-001"
	testName2 := "cloud-nuke-endpoint-002"

	mock := &mockEC2EndpointsClient{
		DescribeVpcEndpointsOutput: ec2.DescribeVpcEndpointsOutput{
			VpcEndpoints: []types.VpcEndpoint{
				{
					VpcEndpointId: aws.String(endpoint1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcEndpointId: aws.String(endpoint2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour)))},
					},
				},
			},
		},
	}

	ids, err := listEC2Endpoints(ctx, mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{endpoint1, endpoint2}, aws.ToStringSlice(ids))
}

func TestListEC2Endpoints_WithNameExclusionFilter(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	now := time.Now()
	endpoint1 := "vpce-0b201b2dcd4f77a2f001"
	endpoint2 := "vpce-0b201b2dcd4f77a2f002"
	testName1 := "cloud-nuke-endpoint-001"
	testName2 := "cloud-nuke-endpoint-002"

	mock := &mockEC2EndpointsClient{
		DescribeVpcEndpointsOutput: ec2.DescribeVpcEndpointsOutput{
			VpcEndpoints: []types.VpcEndpoint{
				{
					VpcEndpointId: aws.String(endpoint1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcEndpointId: aws.String(endpoint2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour)))},
					},
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
		},
	}

	ids, err := listEC2Endpoints(ctx, mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{endpoint2}, aws.ToStringSlice(ids))
}

func TestListEC2Endpoints_WithTimeAfterExclusionFilter(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	now := time.Now()
	endpoint1 := "vpce-0b201b2dcd4f77a2f001"
	endpoint2 := "vpce-0b201b2dcd4f77a2f002"
	testName1 := "cloud-nuke-endpoint-001"
	testName2 := "cloud-nuke-endpoint-002"

	mock := &mockEC2EndpointsClient{
		DescribeVpcEndpointsOutput: ec2.DescribeVpcEndpointsOutput{
			VpcEndpoints: []types.VpcEndpoint{
				{
					VpcEndpointId: aws.String(endpoint1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcEndpointId: aws.String(endpoint2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour)))},
					},
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now),
		},
	}

	ids, err := listEC2Endpoints(ctx, mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{endpoint1}, aws.ToStringSlice(ids))
}

func TestDeleteEC2Endpoint(t *testing.T) {
	t.Parallel()

	mock := &mockEC2EndpointsClient{}
	err := deleteEC2Endpoint(context.Background(), mock, aws.String("vpce-12345"))
	require.NoError(t, err)
}

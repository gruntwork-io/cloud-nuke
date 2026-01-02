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

type mockEC2EndpointsClient struct {
	DescribeVpcEndpointsOutput ec2.DescribeVpcEndpointsOutput
	DeleteVpcEndpointsOutput   ec2.DeleteVpcEndpointsOutput
	DescribeVpcsOutput         ec2.DescribeVpcsOutput
}

func (m *mockEC2EndpointsClient) DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	return &m.DescribeVpcEndpointsOutput, nil
}

func (m *mockEC2EndpointsClient) DeleteVpcEndpoints(ctx context.Context, params *ec2.DeleteVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointsOutput, error) {
	return &m.DeleteVpcEndpointsOutput, nil
}

func (m *mockEC2EndpointsClient) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func TestListEC2Endpoints(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	now := time.Now()

	endpoint1 := "vpce-001"
	endpoint2 := "vpce-002"
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

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{endpoint1, endpoint2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{endpoint2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{endpoint1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listEC2Endpoints(ctx, mock, resource.Scope{}, tc.configObj, false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteEC2Endpoints(t *testing.T) {
	t.Parallel()

	mock := &mockEC2EndpointsClient{}
	err := deleteEC2Endpoints(context.Background(), mock, []string{"vpce-12345", "vpce-67890"})
	require.NoError(t, err)
}

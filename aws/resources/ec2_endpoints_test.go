package resources

import (
	"context"
	"fmt"
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
	DescribeVpcEndpointsOutput  ec2.DescribeVpcEndpointsOutput
	DescribeVpcEndpointsOutputs []ec2.DescribeVpcEndpointsOutput
	DescribeVpcEndpointsError   error
	describeCalls               int
	DeleteVpcEndpointsOutput    ec2.DeleteVpcEndpointsOutput
	DeleteVpcEndpointsError     error
	DescribeVpcsOutput          ec2.DescribeVpcsOutput
}

func (m *mockEC2EndpointsClient) DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	if len(m.DescribeVpcEndpointsOutputs) > 0 && len(params.Filters) > 0 {
		idx := m.describeCalls
		m.describeCalls++
		if idx >= len(m.DescribeVpcEndpointsOutputs) {
			idx = len(m.DescribeVpcEndpointsOutputs) - 1
		}
		return &m.DescribeVpcEndpointsOutputs[idx], nil
	}
	if len(params.Filters) > 0 && m.DescribeVpcEndpointsError != nil {
		return nil, m.DescribeVpcEndpointsError
	}
	return &m.DescribeVpcEndpointsOutput, nil
}

func (m *mockEC2EndpointsClient) DeleteVpcEndpoints(ctx context.Context, params *ec2.DeleteVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointsOutput, error) {
	if m.DeleteVpcEndpointsError != nil {
		return nil, m.DeleteVpcEndpointsError
	}
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

	t.Run("requesterManagedFilter", func(t *testing.T) {
		reqManagedEndpoint := "vpce-reqmanaged"
		nilReqManagedEndpoint := "vpce-nilreqmanaged"
		mockWithReqManaged := &mockEC2EndpointsClient{
			DescribeVpcEndpointsOutput: ec2.DescribeVpcEndpointsOutput{
				VpcEndpoints: []types.VpcEndpoint{
					{
						VpcEndpointId:    aws.String(endpoint1),
						RequesterManaged: aws.Bool(false),
						Tags: []types.Tag{
							{Key: aws.String("Name"), Value: aws.String(testName1)},
							{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
						},
					},
					{
						VpcEndpointId:    aws.String(reqManagedEndpoint),
						RequesterManaged: aws.Bool(true),
						Tags: []types.Tag{
							{Key: aws.String("Name"), Value: aws.String("requester-managed-ep")},
							{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
						},
					},
					{
						// RequesterManaged nil — should be treated as user-managed and included
						VpcEndpointId: aws.String(nilReqManagedEndpoint),
						Tags: []types.Tag{
							{Key: aws.String("Name"), Value: aws.String("nil-requester-managed-ep")},
							{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
						},
					},
				},
			},
		}
		ids, err := listEC2Endpoints(ctx, mockWithReqManaged, resource.Scope{}, config.ResourceType{}, false)
		require.NoError(t, err)
		require.Equal(t, []string{endpoint1, nilReqManagedEndpoint}, aws.ToStringSlice(ids))
	})

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listEC2Endpoints(ctx, mock, resource.Scope{}, tc.configObj, false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteEC2Endpoint(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		mock := &mockEC2EndpointsClient{}
		require.NoError(t, deleteEC2Endpoint(context.Background(), mock, aws.String("vpce-12345")))
	})

	t.Run("unsuccessful item", func(t *testing.T) {
		mock := &mockEC2EndpointsClient{
			DeleteVpcEndpointsOutput: ec2.DeleteVpcEndpointsOutput{
				Unsuccessful: []types.UnsuccessfulItem{{
					ResourceId: aws.String("vpce-12345"),
					Error:      &types.UnsuccessfulItemError{Message: aws.String("endpoint is in use")},
				}},
			},
		}
		err := deleteEC2Endpoint(context.Background(), mock, aws.String("vpce-12345"))
		require.ErrorContains(t, err, "endpoint is in use")
	})

	t.Run("unsuccessful item with nil error", func(t *testing.T) {
		mock := &mockEC2EndpointsClient{
			DeleteVpcEndpointsOutput: ec2.DeleteVpcEndpointsOutput{
				Unsuccessful: []types.UnsuccessfulItem{{ResourceId: aws.String("vpce-12345")}},
			},
		}
		err := deleteEC2Endpoint(context.Background(), mock, aws.String("vpce-12345"))
		require.ErrorContains(t, err, "unknown error")
	})

	t.Run("api error", func(t *testing.T) {
		mock := &mockEC2EndpointsClient{DeleteVpcEndpointsError: fmt.Errorf("api error")}
		require.Error(t, deleteEC2Endpoint(context.Background(), mock, aws.String("vpce-12345")))
	})
}

func TestWaitForEndpointsDeleted(t *testing.T) {
	t.Parallel()

	t.Run("already deleted", func(t *testing.T) {
		mock := &mockEC2EndpointsClient{
			DescribeVpcEndpointsOutputs: []ec2.DescribeVpcEndpointsOutput{
				{VpcEndpoints: []types.VpcEndpoint{}},
			},
		}
		require.NoError(t, waitForEndpointsDeleted(context.Background(), mock, []string{"vpce-12345"}))
	})

	t.Run("deleting then deleted", func(t *testing.T) {
		mock := &mockEC2EndpointsClient{
			DescribeVpcEndpointsOutputs: []ec2.DescribeVpcEndpointsOutput{
				{VpcEndpoints: []types.VpcEndpoint{{VpcEndpointId: aws.String("vpce-12345"), State: "deleting"}}},
				{VpcEndpoints: []types.VpcEndpoint{}},
			},
		}
		require.NoError(t, waitForEndpointsDeleted(context.Background(), mock, []string{"vpce-12345"}))
		require.GreaterOrEqual(t, mock.describeCalls, 2)
	})

	t.Run("ignores unrelated endpoints", func(t *testing.T) {
		mock := &mockEC2EndpointsClient{
			DescribeVpcEndpointsOutputs: []ec2.DescribeVpcEndpointsOutput{
				// Unrelated endpoint still deleting — ours is gone
				{VpcEndpoints: []types.VpcEndpoint{{VpcEndpointId: aws.String("vpce-other"), State: "deleting"}}},
			},
		}
		require.NoError(t, waitForEndpointsDeleted(context.Background(), mock, []string{"vpce-12345"}))
	})

	t.Run("api error is fatal", func(t *testing.T) {
		mock := &mockEC2EndpointsClient{
			DescribeVpcEndpointsError: fmt.Errorf("throttling"),
		}
		require.Error(t, waitForEndpointsDeleted(context.Background(), mock, []string{"vpce-12345"}))
		require.Equal(t, 0, mock.describeCalls, "FatalError should stop retries after first attempt")
	})
}

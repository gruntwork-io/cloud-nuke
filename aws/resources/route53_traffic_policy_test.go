package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockRoute53TrafficPolicyClient struct {
	ListTrafficPoliciesOutput  route53.ListTrafficPoliciesOutput
	DeleteTrafficPolicyOutput  route53.DeleteTrafficPolicyOutput
	DeleteTrafficPolicyError   error
	ListTrafficPoliciesOutputs []route53.ListTrafficPoliciesOutput
	listCallCount              int
}

func (m *mockRoute53TrafficPolicyClient) ListTrafficPolicies(_ context.Context, _ *route53.ListTrafficPoliciesInput, _ ...func(*route53.Options)) (*route53.ListTrafficPoliciesOutput, error) {
	if len(m.ListTrafficPoliciesOutputs) > 0 {
		if m.listCallCount < len(m.ListTrafficPoliciesOutputs) {
			output := m.ListTrafficPoliciesOutputs[m.listCallCount]
			m.listCallCount++
			return &output, nil
		}
	}
	return &m.ListTrafficPoliciesOutput, nil
}

func (m *mockRoute53TrafficPolicyClient) DeleteTrafficPolicy(_ context.Context, _ *route53.DeleteTrafficPolicyInput, _ ...func(*route53.Options)) (*route53.DeleteTrafficPolicyOutput, error) {
	return &m.DeleteTrafficPolicyOutput, m.DeleteTrafficPolicyError
}

func TestListRoute53TrafficPolicies(t *testing.T) {
	t.Parallel()

	testId1 := "d8c6f2db-89dd-5533-f30c-13e28eba8818"
	testId2 := "d8c6f2db-90dd-5533-f30c-13e28eba8818"
	testName1 := "Test name 01"
	testName2 := "Test name 02"

	mock := &mockRoute53TrafficPolicyClient{
		ListTrafficPoliciesOutput: route53.ListTrafficPoliciesOutput{
			TrafficPolicySummaries: []types.TrafficPolicySummary{
				{
					Id:            aws.String(testId1),
					Name:          aws.String(testName1),
					LatestVersion: aws.Int32(1),
				},
				{
					Id:            aws.String(testId2),
					Name:          aws.String(testName2),
					LatestVersion: aws.Int32(2),
				},
			},
			IsTruncated: false,
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			// Format: "id:version"
			expected: []string{testId1 + ":1", testId2 + ":2"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testId2 + ":2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listRoute53TrafficPolicies(context.Background(), mock, resource.Scope{Region: "global"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestListRoute53TrafficPolicies_Pagination(t *testing.T) {
	t.Parallel()

	testId1 := "policy-1"
	testId2 := "policy-2"

	mock := &mockRoute53TrafficPolicyClient{
		ListTrafficPoliciesOutputs: []route53.ListTrafficPoliciesOutput{
			{
				TrafficPolicySummaries: []types.TrafficPolicySummary{
					{Id: aws.String(testId1), Name: aws.String("Policy 1"), LatestVersion: aws.Int32(1)},
				},
				IsTruncated:           true,
				TrafficPolicyIdMarker: aws.String(testId1),
			},
			{
				TrafficPolicySummaries: []types.TrafficPolicySummary{
					{Id: aws.String(testId2), Name: aws.String("Policy 2"), LatestVersion: aws.Int32(3)},
				},
				IsTruncated: false,
			},
		},
	}

	ids, err := listRoute53TrafficPolicies(context.Background(), mock, resource.Scope{Region: "global"}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{testId1 + ":1", testId2 + ":3"}, aws.ToStringSlice(ids))
}

func TestDeleteRoute53TrafficPolicy(t *testing.T) {
	t.Parallel()

	mock := &mockRoute53TrafficPolicyClient{}
	// Identifier format: "id:version"
	err := deleteRoute53TrafficPolicy(context.Background(), mock, aws.String("policy-to-delete:1"))
	require.NoError(t, err)
}

func TestDeleteRoute53TrafficPolicy_InvalidFormat(t *testing.T) {
	t.Parallel()

	mock := &mockRoute53TrafficPolicyClient{}
	// Invalid format - missing version
	err := deleteRoute53TrafficPolicy(context.Background(), mock, aws.String("policy-without-version"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid traffic policy identifier format")
}

func TestParseTrafficPolicyIdentifier(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		identifier      string
		expectedId      string
		expectedVersion int32
		expectError     bool
	}{
		"valid identifier": {
			identifier:      "abc123:5",
			expectedId:      "abc123",
			expectedVersion: 5,
			expectError:     false,
		},
		"missing version": {
			identifier:  "abc123",
			expectError: true,
		},
		"invalid version": {
			identifier:  "abc123:notanumber",
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			id, version, err := parseTrafficPolicyIdentifier(tc.identifier)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedId, id)
				require.Equal(t, tc.expectedVersion, version)
			}
		})
	}
}

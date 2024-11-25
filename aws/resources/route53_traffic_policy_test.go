package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedR53TrafficPolicy struct {
	Route53TrafficPolicyAPI
	ListTrafficPoliciesOutput route53.ListTrafficPoliciesOutput
	DeleteTrafficPolicyOutput route53.DeleteTrafficPolicyOutput
}

func (mock mockedR53TrafficPolicy) ListTrafficPolicies(_ context.Context, _ *route53.ListTrafficPoliciesInput, _ ...func(*route53.Options)) (*route53.ListTrafficPoliciesOutput, error) {
	return &mock.ListTrafficPoliciesOutput, nil
}

func (mock mockedR53TrafficPolicy) DeleteTrafficPolicy(_ context.Context, _ *route53.DeleteTrafficPolicyInput, _ ...func(*route53.Options)) (*route53.DeleteTrafficPolicyOutput, error) {
	return &mock.DeleteTrafficPolicyOutput, nil
}

func TestR53TrafficPolicy_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "d8c6f2db-89dd-5533-f30c-13e28eba8818"
	testId2 := "d8c6f2db-90dd-5533-f30c-13e28eba8818"

	testName1 := "Test name 01"
	testName2 := "Test name 02"
	rc := Route53TrafficPolicy{
		Client: mockedR53TrafficPolicy{
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
						LatestVersion: aws.Int32(1),
					},
				},
			},
		},
		versionMap: make(map[string]*int32),
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := rc.getAll(context.Background(), config.Config{
				Route53TrafficPolicy: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestR53TrafficPolicy_Nuke(t *testing.T) {

	t.Parallel()

	rc := Route53TrafficPolicy{
		Client: mockedR53TrafficPolicy{
			DeleteTrafficPolicyOutput: route53.DeleteTrafficPolicyOutput{},
		},
		versionMap: make(map[string]*int32),
	}

	err := rc.nukeAll([]*string{aws.String("policy-01")})
	require.NoError(t, err)
}

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

type mockedR53HostedZone struct {
	Route53HostedZoneAPI
	ListResourceRecordSetsOutput      route53.ListResourceRecordSetsOutput
	ChangeResourceRecordSetsOutput    route53.ChangeResourceRecordSetsOutput
	ListHostedZonesOutput             route53.ListHostedZonesOutput
	DeleteHostedZoneOutput            route53.DeleteHostedZoneOutput
	DeleteTrafficPolicyInstanceOutput route53.DeleteTrafficPolicyInstanceOutput
}

func (mock mockedR53HostedZone) ListHostedZones(_ context.Context, _ *route53.ListHostedZonesInput, _ ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error) {
	return &mock.ListHostedZonesOutput, nil
}

func (mock mockedR53HostedZone) ListResourceRecordSets(_ context.Context, _ *route53.ListResourceRecordSetsInput, _ ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
	return &mock.ListResourceRecordSetsOutput, nil
}

func (mock mockedR53HostedZone) ChangeResourceRecordSets(_ context.Context, _ *route53.ChangeResourceRecordSetsInput, _ ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
	return &mock.ChangeResourceRecordSetsOutput, nil
}

func (mock mockedR53HostedZone) DeleteHostedZone(_ context.Context, _ *route53.DeleteHostedZoneInput, _ ...func(*route53.Options)) (*route53.DeleteHostedZoneOutput, error) {
	return &mock.DeleteHostedZoneOutput, nil
}

func (mock mockedR53HostedZone) DeleteTrafficPolicyInstance(_ context.Context, _ *route53.DeleteTrafficPolicyInstanceInput, _ ...func(*route53.Options)) (*route53.DeleteTrafficPolicyInstanceOutput, error) {
	return &mock.DeleteTrafficPolicyInstanceOutput, nil
}

func TestR53HostedZone_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "d8c6f2db-89dd-5533-f30c-13e28eba8818"
	testId2 := "d8c6f2db-90dd-5533-f30c-13e28eba8818"

	testName1 := "Test name 01"
	testName2 := "Test name 02"
	rc := Route53HostedZone{
		HostedZonesDomains: map[string]*types.HostedZone{
			testId1: {
				Name: aws.String(testName1),
			},
			testId2: {
				Name: aws.String(testName2),
			},
		},
		Client: mockedR53HostedZone{
			ListHostedZonesOutput: route53.ListHostedZonesOutput{
				HostedZones: []types.HostedZone{
					{
						Id:   aws.String(testId1),
						Name: aws.String(testName1),
					},
					{
						Id:   aws.String(testId2),
						Name: aws.String(testName2),
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
				Route53HostedZone: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestR53HostedZone_Nuke(t *testing.T) {

	t.Parallel()

	rc := Route53HostedZone{
		HostedZonesDomains: map[string]*types.HostedZone{
			"collection-id-01": {
				Name: aws.String("domain.com"),
			},
		},
		Client: mockedR53HostedZone{
			DeleteHostedZoneOutput: route53.DeleteHostedZoneOutput{},
		},
	}

	err := rc.nukeAll([]*string{aws.String("collection-id-01")})
	require.NoError(t, err)
}

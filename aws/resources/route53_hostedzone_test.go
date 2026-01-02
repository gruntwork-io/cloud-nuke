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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedRoute53HostedZone struct {
	Route53HostedZoneAPI
	ListHostedZonesOutput             route53.ListHostedZonesOutput
	ListTagsForResourcesOutput        route53.ListTagsForResourcesOutput
	ListResourceRecordSetsOutput      route53.ListResourceRecordSetsOutput
	ChangeResourceRecordSetsOutput    route53.ChangeResourceRecordSetsOutput
	DeleteHostedZoneOutput            route53.DeleteHostedZoneOutput
	DeleteTrafficPolicyInstanceOutput route53.DeleteTrafficPolicyInstanceOutput
}

func (m mockedRoute53HostedZone) ListHostedZones(_ context.Context, _ *route53.ListHostedZonesInput, _ ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error) {
	return &m.ListHostedZonesOutput, nil
}

func (m mockedRoute53HostedZone) ListTagsForResources(_ context.Context, params *route53.ListTagsForResourcesInput, _ ...func(*route53.Options)) (*route53.ListTagsForResourcesOutput, error) {
	// Filter the output to only include requested resource IDs
	var filtered []types.ResourceTagSet
	for _, tagSet := range m.ListTagsForResourcesOutput.ResourceTagSets {
		for _, id := range params.ResourceIds {
			if aws.ToString(tagSet.ResourceId) == id {
				filtered = append(filtered, tagSet)
				break
			}
		}
	}
	return &route53.ListTagsForResourcesOutput{ResourceTagSets: filtered}, nil
}

func (m mockedRoute53HostedZone) ListResourceRecordSets(_ context.Context, _ *route53.ListResourceRecordSetsInput, _ ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
	return &m.ListResourceRecordSetsOutput, nil
}

func (m mockedRoute53HostedZone) ChangeResourceRecordSets(_ context.Context, _ *route53.ChangeResourceRecordSetsInput, _ ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
	return &m.ChangeResourceRecordSetsOutput, nil
}

func (m mockedRoute53HostedZone) DeleteHostedZone(_ context.Context, _ *route53.DeleteHostedZoneInput, _ ...func(*route53.Options)) (*route53.DeleteHostedZoneOutput, error) {
	return &m.DeleteHostedZoneOutput, nil
}

func (m mockedRoute53HostedZone) DeleteTrafficPolicyInstance(_ context.Context, _ *route53.DeleteTrafficPolicyInstanceInput, _ ...func(*route53.Options)) (*route53.DeleteTrafficPolicyInstanceOutput, error) {
	return &m.DeleteTrafficPolicyInstanceOutput, nil
}

func TestRoute53HostedZone_List(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		hostedZones []types.HostedZone
		tagSets     []types.ResourceTagSet
		configObj   config.ResourceType
		expected    []string
	}{
		"empty filter returns all zones": {
			hostedZones: []types.HostedZone{
				{Id: aws.String("/hostedzone/zone1"), Name: aws.String("example1.com.")},
				{Id: aws.String("/hostedzone/zone2"), Name: aws.String("example2.com.")},
			},
			tagSets: []types.ResourceTagSet{
				{ResourceId: aws.String("zone1"), Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}}},
				{ResourceId: aws.String("zone2"), Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}}},
			},
			configObj: config.ResourceType{},
			// Format: "zoneId|domainName"
			expected: []string{"/hostedzone/zone1|example1.com.", "/hostedzone/zone2|example2.com."},
		},
		"name exclusion filter": {
			hostedZones: []types.HostedZone{
				{Id: aws.String("/hostedzone/zone1"), Name: aws.String("example1.com.")},
				{Id: aws.String("/hostedzone/zone2"), Name: aws.String("example2.com.")},
			},
			tagSets: []types.ResourceTagSet{
				{ResourceId: aws.String("zone1"), Tags: []types.Tag{}},
				{ResourceId: aws.String("zone2"), Tags: []types.Tag{}},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("example1")}},
				},
			},
			expected: []string{"/hostedzone/zone2|example2.com."},
		},
		"tag exclusion filter": {
			hostedZones: []types.HostedZone{
				{Id: aws.String("/hostedzone/zone1"), Name: aws.String("example1.com.")},
				{Id: aws.String("/hostedzone/zone2"), Name: aws.String("example2.com.")},
			},
			tagSets: []types.ResourceTagSet{
				{ResourceId: aws.String("zone1"), Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}}},
				{ResourceId: aws.String("zone2"), Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}}},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("prod")}},
				},
			},
			expected: []string{"/hostedzone/zone2|example2.com."},
		},
		"tag inclusion filter": {
			hostedZones: []types.HostedZone{
				{Id: aws.String("/hostedzone/zone1"), Name: aws.String("example1.com.")},
				{Id: aws.String("/hostedzone/zone2"), Name: aws.String("example2.com.")},
			},
			tagSets: []types.ResourceTagSet{
				{ResourceId: aws.String("zone1"), Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}}},
				{ResourceId: aws.String("zone2"), Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}}},
			},
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("prod")}},
				},
			},
			expected: []string{"/hostedzone/zone1|example1.com."},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockedRoute53HostedZone{
				ListHostedZonesOutput:      route53.ListHostedZonesOutput{HostedZones: tc.hostedZones},
				ListTagsForResourcesOutput: route53.ListTagsForResourcesOutput{ResourceTagSets: tc.tagSets},
			}

			ids, err := listRoute53HostedZones(context.Background(), client, resource.Scope{Region: "global"}, tc.configObj)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestRoute53HostedZone_Delete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		recordSets []types.ResourceRecordSet
		expectErr  bool
	}{
		"delete zone with no custom records": {
			recordSets: []types.ResourceRecordSet{
				{Name: aws.String("example.com."), Type: types.RRTypeSoa, TTL: aws.Int64(300)},
				{Name: aws.String("example.com."), Type: types.RRTypeNs, TTL: aws.Int64(300)},
			},
			expectErr: false,
		},
		"delete zone with custom A record": {
			recordSets: []types.ResourceRecordSet{
				{Name: aws.String("example.com."), Type: types.RRTypeSoa, TTL: aws.Int64(300)},
				{Name: aws.String("example.com."), Type: types.RRTypeNs, TTL: aws.Int64(300)},
				{Name: aws.String("www.example.com."), Type: types.RRTypeA, TTL: aws.Int64(300), ResourceRecords: []types.ResourceRecord{{Value: aws.String("1.2.3.4")}}},
			},
			expectErr: false,
		},
		"delete zone with traffic policy record": {
			recordSets: []types.ResourceRecordSet{
				{Name: aws.String("example.com."), Type: types.RRTypeSoa, TTL: aws.Int64(300)},
				{Name: aws.String("example.com."), Type: types.RRTypeNs, TTL: aws.Int64(300)},
				{Name: aws.String("api.example.com."), Type: types.RRTypeA, TrafficPolicyInstanceId: aws.String("tp-123")},
			},
			expectErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockedRoute53HostedZone{
				ListResourceRecordSetsOutput:      route53.ListResourceRecordSetsOutput{ResourceRecordSets: tc.recordSets},
				ChangeResourceRecordSetsOutput:    route53.ChangeResourceRecordSetsOutput{},
				DeleteHostedZoneOutput:            route53.DeleteHostedZoneOutput{},
				DeleteTrafficPolicyInstanceOutput: route53.DeleteTrafficPolicyInstanceOutput{},
			}

			// Identifier format: "zoneId|domainName"
			err := deleteRoute53HostedZone(context.Background(), client, aws.String("/hostedzone/test-zone|example.com."))
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseHostedZoneIdentifier(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		identifier     string
		expectedId     string
		expectedDomain string
		expectError    bool
	}{
		"valid identifier": {
			identifier:     "/hostedzone/abc123|example.com.",
			expectedId:     "/hostedzone/abc123",
			expectedDomain: "example.com.",
			expectError:    false,
		},
		"missing domain": {
			identifier:  "/hostedzone/abc123",
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			id, domain, err := parseHostedZoneIdentifier(tc.identifier)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedId, id)
				require.Equal(t, tc.expectedDomain, domain)
			}
		})
	}
}

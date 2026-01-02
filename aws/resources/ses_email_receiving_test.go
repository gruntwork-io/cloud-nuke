package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

// Mock for SES Receipt Rule Set API
type mockSESReceiptRuleSetAPI struct {
	SESReceiptRuleSetAPI
	DescribeActiveReceiptRuleSetOutput ses.DescribeActiveReceiptRuleSetOutput
	ListReceiptRuleSetsOutput          ses.ListReceiptRuleSetsOutput
	DeleteReceiptRuleSetOutput         ses.DeleteReceiptRuleSetOutput
}

func (m mockSESReceiptRuleSetAPI) DescribeActiveReceiptRuleSet(ctx context.Context, params *ses.DescribeActiveReceiptRuleSetInput, optFns ...func(*ses.Options)) (*ses.DescribeActiveReceiptRuleSetOutput, error) {
	return &m.DescribeActiveReceiptRuleSetOutput, nil
}

func (m mockSESReceiptRuleSetAPI) ListReceiptRuleSets(ctx context.Context, params *ses.ListReceiptRuleSetsInput, optFns ...func(*ses.Options)) (*ses.ListReceiptRuleSetsOutput, error) {
	return &m.ListReceiptRuleSetsOutput, nil
}

func (m mockSESReceiptRuleSetAPI) DeleteReceiptRuleSet(ctx context.Context, params *ses.DeleteReceiptRuleSetInput, optFns ...func(*ses.Options)) (*ses.DeleteReceiptRuleSetOutput, error) {
	return &m.DeleteReceiptRuleSetOutput, nil
}

func TestListSesReceiptRuleSets(t *testing.T) {
	t.Parallel()

	now := time.Now()
	id1, id2 := "test-ruleset-1", "test-ruleset-2"

	tests := map[string]struct {
		region    string
		ruleSets  []types.ReceiptRuleSetMetadata
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			region: "us-east-1",
			ruleSets: []types.ReceiptRuleSetMetadata{
				{Name: aws.String(id1), CreatedTimestamp: aws.Time(now)},
				{Name: aws.String(id2), CreatedTimestamp: aws.Time(now.AddDate(-1, 0, 0))},
			},
			configObj: config.ResourceType{},
			expected:  []string{id1, id2},
		},
		"nameExclusionFilter": {
			region: "us-east-1",
			ruleSets: []types.ReceiptRuleSetMetadata{
				{Name: aws.String(id1), CreatedTimestamp: aws.Time(now)},
				{Name: aws.String(id2), CreatedTimestamp: aws.Time(now.AddDate(-1, 0, 0))},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(id2)}},
				},
			},
			expected: []string{id1},
		},
		"timeAfterExclusionFilter": {
			region: "us-east-1",
			ruleSets: []types.ReceiptRuleSetMetadata{
				{Name: aws.String(id1), CreatedTimestamp: aws.Time(now)},
				{Name: aws.String(id2), CreatedTimestamp: aws.Time(now.AddDate(-1, 0, 0))},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{id2},
		},
		"unsupportedRegion": {
			region:    "us-west-1",
			ruleSets:  []types.ReceiptRuleSetMetadata{{Name: aws.String(id1)}},
			configObj: config.ResourceType{},
			expected:  []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockSESReceiptRuleSetAPI{
				ListReceiptRuleSetsOutput: ses.ListReceiptRuleSetsOutput{RuleSets: tc.ruleSets},
			}
			scope := resource.Scope{Region: tc.region}

			names, err := listSesReceiptRuleSets(context.Background(), client, scope, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteSesReceiptRuleSet(t *testing.T) {
	t.Parallel()

	client := mockSESReceiptRuleSetAPI{}
	err := deleteSesReceiptRuleSet(context.Background(), client, aws.String("test-ruleset"))
	require.NoError(t, err)
}

// Mock for SES Receipt Filter API
type mockSESReceiptFilterAPI struct {
	SESReceiptFilterAPI
	ListReceiptFiltersOutput  ses.ListReceiptFiltersOutput
	DeleteReceiptFilterOutput ses.DeleteReceiptFilterOutput
}

func (m mockSESReceiptFilterAPI) ListReceiptFilters(ctx context.Context, params *ses.ListReceiptFiltersInput, optFns ...func(*ses.Options)) (*ses.ListReceiptFiltersOutput, error) {
	return &m.ListReceiptFiltersOutput, nil
}

func (m mockSESReceiptFilterAPI) DeleteReceiptFilter(ctx context.Context, params *ses.DeleteReceiptFilterInput, optFns ...func(*ses.Options)) (*ses.DeleteReceiptFilterOutput, error) {
	return &m.DeleteReceiptFilterOutput, nil
}

func TestListSesReceiptFilters(t *testing.T) {
	t.Parallel()

	id1, id2 := "test-filter-1", "test-filter-2"

	tests := map[string]struct {
		filters   []types.ReceiptFilter
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			filters: []types.ReceiptFilter{
				{Name: aws.String(id1)},
				{Name: aws.String(id2)},
			},
			configObj: config.ResourceType{},
			expected:  []string{id1, id2},
		},
		"nameExclusionFilter": {
			filters: []types.ReceiptFilter{
				{Name: aws.String(id1)},
				{Name: aws.String(id2)},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(id2)}},
				},
			},
			expected: []string{id1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockSESReceiptFilterAPI{
				ListReceiptFiltersOutput: ses.ListReceiptFiltersOutput{Filters: tc.filters},
			}
			scope := resource.Scope{Region: "us-east-1"}

			names, err := listSesReceiptFilters(context.Background(), client, scope, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteSesReceiptFilter(t *testing.T) {
	t.Parallel()

	client := mockSESReceiptFilterAPI{}
	err := deleteSesReceiptFilter(context.Background(), client, aws.String("test-filter"))
	require.NoError(t, err)
}

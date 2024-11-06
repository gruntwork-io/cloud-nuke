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
	"github.com/stretchr/testify/require"
)

type mockedSesReceiptRule struct {
	SESEmailReceivingAPI
	DeleteReceiptRuleSetOutput         ses.DeleteReceiptRuleSetOutput
	ListReceiptRuleSetsOutput          ses.ListReceiptRuleSetsOutput
	DescribeActiveReceiptRuleSetOutput ses.DescribeActiveReceiptRuleSetOutput
}

func (m mockedSesReceiptRule) ListReceiptRuleSets(ctx context.Context, params *ses.ListReceiptRuleSetsInput, optFns ...func(*ses.Options)) (*ses.ListReceiptRuleSetsOutput, error) {
	return &m.ListReceiptRuleSetsOutput, nil
}

func (m mockedSesReceiptRule) DeleteReceiptRuleSet(ctx context.Context, params *ses.DeleteReceiptRuleSetInput, optFns ...func(*ses.Options)) (*ses.DeleteReceiptRuleSetOutput, error) {
	return &m.DeleteReceiptRuleSetOutput, nil
}

func (m mockedSesReceiptRule) DescribeActiveReceiptRuleSet(ctx context.Context, params *ses.DescribeActiveReceiptRuleSetInput, optFns ...func(*ses.Options)) (*ses.DescribeActiveReceiptRuleSetOutput, error) {
	return &m.DescribeActiveReceiptRuleSetOutput, nil
}

func TestSesReceiptRule_GetAll(t *testing.T) {

	id1 := "test-id-1"
	id2 := "test-id-2"
	metadata1 := types.ReceiptRuleSetMetadata{
		CreatedTimestamp: aws.Time(time.Now()),
		Name:             aws.String(id1),
	}
	metadata2 := types.ReceiptRuleSetMetadata{
		CreatedTimestamp: aws.Time(time.Now().AddDate(-1, 0, 0)),
		Name:             aws.String(id2),
	}
	t.Parallel()

	sesRule := SesReceiptRule{
		Region: "us-east-1",
		Client: mockedSesReceiptRule{
			ListReceiptRuleSetsOutput: ses.ListReceiptRuleSetsOutput{
				RuleSets: []types.ReceiptRuleSetMetadata{
					metadata1,
					metadata2,
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
			expected:  []string{id1, id2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(id2),
					}}},
			},
			expected: []string{id1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(time.Now().Add(-1 * time.Hour)),
				}},
			expected: []string{id2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := sesRule.getAll(context.Background(), config.Config{
				SESReceiptRuleSet: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestSesReceiptRule_NukeAll(t *testing.T) {
	t.Parallel()

	sesRule := SesReceiptRule{
		Client: mockedSesReceiptRule{},
	}

	err := sesRule.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

// //////////////receipt Ip filters///////////////////////////

type mockedSesReceiptFilter struct {
	SESEmailReceivingAPI
	DeleteReceiptFilterOutput ses.DeleteReceiptFilterOutput
	ListReceiptFiltersOutput  ses.ListReceiptFiltersOutput
}

func (m mockedSesReceiptFilter) ListReceiptFilters(ctx context.Context, params *ses.ListReceiptFiltersInput, optFns ...func(*ses.Options)) (*ses.ListReceiptFiltersOutput, error) {
	return &m.ListReceiptFiltersOutput, nil
}

func (m mockedSesReceiptFilter) DeleteReceiptFilter(ctx context.Context, params *ses.DeleteReceiptFilterInput, optFns ...func(*ses.Options)) (*ses.DeleteReceiptFilterOutput, error) {
	return &m.DeleteReceiptFilterOutput, nil
}

func TestSesReceiptFilter_GetAll(t *testing.T) {

	id1 := "test-id-1"
	id2 := "test-id-2"
	metadata1 := types.ReceiptFilter{
		Name: aws.String(id1),
	}
	metadata2 := types.ReceiptFilter{
		Name: aws.String(id2),
	}
	t.Parallel()

	sesRule := SesReceiptFilter{
		Region: "us-east-1",
		Client: mockedSesReceiptFilter{
			ListReceiptFiltersOutput: ses.ListReceiptFiltersOutput{
				Filters: []types.ReceiptFilter{
					metadata1, metadata2,
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
			expected:  []string{id1, id2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(id2),
					}}},
			},
			expected: []string{id1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := sesRule.getAll(context.Background(), config.Config{
				SESReceiptFilter: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}
func TestSesReceiptFilter_NukeAll(t *testing.T) {
	t.Parallel()

	sesRule := SesReceiptFilter{
		Client: mockedSesReceiptFilter{},
	}

	err := sesRule.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedAccessAnalyzer struct {
	AccessAnalyzerAPI
	ListAnalyzersOutput  accessanalyzer.ListAnalyzersOutput
	DeleteAnalyzerOutput accessanalyzer.DeleteAnalyzerOutput
}

func (m mockedAccessAnalyzer) ListAnalyzers(context.Context, *accessanalyzer.ListAnalyzersInput, ...func(*accessanalyzer.Options)) (*accessanalyzer.ListAnalyzersOutput, error) {
	return &m.ListAnalyzersOutput, nil
}

func (m mockedAccessAnalyzer) DeleteAnalyzer(ctx context.Context, params *accessanalyzer.DeleteAnalyzerInput, optFns ...func(*accessanalyzer.Options)) (*accessanalyzer.DeleteAnalyzerOutput, error) {
	return &m.DeleteAnalyzerOutput, nil
}

func TestAccessAnalyzer_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testName1 := "test1"
	testName2 := "test2"
	aa := AccessAnalyzer{
		Client: mockedAccessAnalyzer{
			ListAnalyzersOutput: accessanalyzer.ListAnalyzersOutput{
				Analyzers: []types.AnalyzerSummary{
					{
						Name:      aws.String(testName1),
						CreatedAt: aws.Time(now),
					},
					{
						Name:      aws.String(testName2),
						CreatedAt: aws.Time(now.Add(1)),
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
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := aa.getAll(context.Background(), config.Config{
				AccessAnalyzer: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestAccessAnalyzer_NukeAll(t *testing.T) {
	t.Parallel()

	aa := AccessAnalyzer{
		Client: mockedAccessAnalyzer{
			DeleteAnalyzerOutput: accessanalyzer.DeleteAnalyzerOutput{},
		},
	}

	err := aa.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

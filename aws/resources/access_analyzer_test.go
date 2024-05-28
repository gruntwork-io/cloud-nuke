package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/accessanalyzer"
	"github.com/aws/aws-sdk-go/service/accessanalyzer/accessanalyzeriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedAccessAnalyzer struct {
	accessanalyzeriface.AccessAnalyzerAPI
	ListAnalyzersPagesOutput accessanalyzer.ListAnalyzersOutput
	DeleteAnalyzerOutput     accessanalyzer.DeleteAnalyzerOutput
}

func (m mockedAccessAnalyzer) ListAnalyzersPagesWithContext(_ awsgo.Context, _ *accessanalyzer.ListAnalyzersInput, callback func(*accessanalyzer.ListAnalyzersOutput, bool) bool, _ ...request.Option) error {
	callback(&m.ListAnalyzersPagesOutput, true)
	return nil
}

func (m mockedAccessAnalyzer) DeleteAnalyzerWithContext(awsgo.Context, *accessanalyzer.DeleteAnalyzerInput, ...request.Option) (*accessanalyzer.DeleteAnalyzerOutput, error) {
	return &m.DeleteAnalyzerOutput, nil
}

func TestAccessAnalyzer_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	testName1 := "test1"
	testName2 := "test2"
	aa := AccessAnalyzer{
		Client: mockedAccessAnalyzer{
			ListAnalyzersPagesOutput: accessanalyzer.ListAnalyzersOutput{
				Analyzers: []*accessanalyzer.AnalyzerSummary{
					{
						Name:      awsgo.String(testName1),
						CreatedAt: awsgo.Time(now),
					},
					{
						Name:      awsgo.String(testName2),
						CreatedAt: awsgo.Time(now.Add(1)),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
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

	err := aa.nukeAll([]*string{awsgo.String("test")})
	require.NoError(t, err)
}

package resources

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/accessanalyzer"
	"github.com/aws/aws-sdk-go/service/accessanalyzer/accessanalyzeriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedAccessAnalyzer struct {
	accessanalyzeriface.AccessAnalyzerAPI
	ListAnalyzersPagesOutput accessanalyzer.ListAnalyzersOutput
	DeleteAnalyzerOutput     accessanalyzer.DeleteAnalyzerOutput
}

func (m mockedAccessAnalyzer) ListAnalyzersPages(input *accessanalyzer.ListAnalyzersInput, callback func(*accessanalyzer.ListAnalyzersOutput, bool) bool) error {
	callback(&m.ListAnalyzersPagesOutput, true)
	return nil
}

func (m mockedAccessAnalyzer) DeleteAnalyzer(input *accessanalyzer.DeleteAnalyzerInput) (*accessanalyzer.DeleteAnalyzerOutput, error) {
	return &m.DeleteAnalyzerOutput, nil
}

func TestAccessAnalyzer_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
			names, err := aa.getAll(config.Config{
				AccessAnalyzer: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestAccessAnalyzer_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	aa := AccessAnalyzer{
		Client: mockedAccessAnalyzer{
			DeleteAnalyzerOutput: accessanalyzer.DeleteAnalyzerOutput{},
		},
	}

	err := aa.nukeAll([]*string{awsgo.String("test")})
	require.NoError(t, err)
}

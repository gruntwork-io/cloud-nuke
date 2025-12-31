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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockAccessAnalyzerClient struct {
	ListAnalyzersOutput  accessanalyzer.ListAnalyzersOutput
	DeleteAnalyzerOutput accessanalyzer.DeleteAnalyzerOutput
}

func (m *mockAccessAnalyzerClient) ListAnalyzers(ctx context.Context, params *accessanalyzer.ListAnalyzersInput, optFns ...func(*accessanalyzer.Options)) (*accessanalyzer.ListAnalyzersOutput, error) {
	return &m.ListAnalyzersOutput, nil
}

func (m *mockAccessAnalyzerClient) DeleteAnalyzer(ctx context.Context, params *accessanalyzer.DeleteAnalyzerInput, optFns ...func(*accessanalyzer.Options)) (*accessanalyzer.DeleteAnalyzerOutput, error) {
	return &m.DeleteAnalyzerOutput, nil
}

func TestListAccessAnalyzers(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockAccessAnalyzerClient{
		ListAnalyzersOutput: accessanalyzer.ListAnalyzersOutput{
			Analyzers: []types.AnalyzerSummary{
				{Name: aws.String("analyzer1"), CreatedAt: aws.Time(now)},
				{Name: aws.String("analyzer2"), CreatedAt: aws.Time(now)},
			},
		},
	}

	names, err := listAccessAnalyzers(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"analyzer1", "analyzer2"}, aws.ToStringSlice(names))
}

func TestListAccessAnalyzers_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockAccessAnalyzerClient{
		ListAnalyzersOutput: accessanalyzer.ListAnalyzersOutput{
			Analyzers: []types.AnalyzerSummary{
				{Name: aws.String("analyzer1"), CreatedAt: aws.Time(now)},
				{Name: aws.String("skip-this"), CreatedAt: aws.Time(now)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listAccessAnalyzers(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"analyzer1"}, aws.ToStringSlice(names))
}

func TestDeleteAccessAnalyzer(t *testing.T) {
	t.Parallel()

	mock := &mockAccessAnalyzerClient{}
	err := deleteAccessAnalyzer(context.Background(), mock, aws.String("test-analyzer"))
	require.NoError(t, err)
}

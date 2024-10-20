package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedCloudWatchDashboard struct {
	CloudWatchDashboardsAPI
	ListDashboardsOutput   cloudwatch.ListDashboardsOutput
	DeleteDashboardsOutput cloudwatch.DeleteDashboardsOutput
}

func (m mockedCloudWatchDashboard) ListDashboards(ctx context.Context, params *cloudwatch.ListDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListDashboardsOutput, error) {
	return &m.ListDashboardsOutput, nil
}

func (m mockedCloudWatchDashboard) DeleteDashboards(ctx context.Context, params *cloudwatch.DeleteDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteDashboardsOutput, error) {
	return &m.DeleteDashboardsOutput, nil
}

func TestCloudWatchDashboard_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	cw := CloudWatchDashboards{
		Client: mockedCloudWatchDashboard{
			ListDashboardsOutput: cloudwatch.ListDashboardsOutput{
				DashboardEntries: []types.DashboardEntry{
					{
						DashboardName: aws.String(testName1),
						LastModified:  &now,
					},
					{
						DashboardName: aws.String(testName2),
						LastModified:  aws.Time(now.Add(1)),
					},
				}},
		}}

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
					TimeAfter: aws.Time(now.Add(-1)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := cw.getAll(context.Background(), config.Config{
				CloudWatchDashboard: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestCloudWatchDashboard_NukeAll(t *testing.T) {
	t.Parallel()
	cw := CloudWatchDashboards{
		Client: mockedCloudWatchDashboard{
			DeleteDashboardsOutput: cloudwatch.DeleteDashboardsOutput{},
		}}

	err := cw.nukeAll([]*string{aws.String("test-name1"), aws.String("test-name2")})
	require.NoError(t, err)
}

package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedCloudWatchDashboard struct {
	cloudwatchiface.CloudWatchAPI
	ListDashboardsOutput   cloudwatch.ListDashboardsOutput
	DeleteDashboardsOutput cloudwatch.DeleteDashboardsOutput
}

func (m mockedCloudWatchDashboard) ListDashboardsPages(input *cloudwatch.ListDashboardsInput, fn func(*cloudwatch.ListDashboardsOutput, bool) bool) error {
	fn(&m.ListDashboardsOutput, true)
	return nil
}

func (m mockedCloudWatchDashboard) DeleteDashboards(input *cloudwatch.DeleteDashboardsInput) (*cloudwatch.DeleteDashboardsOutput, error) {
	return &m.DeleteDashboardsOutput, nil
}

func TestCloudWatchDashboard_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	cw := CloudWatchDashboards{
		Client: mockedCloudWatchDashboard{
			ListDashboardsOutput: cloudwatch.ListDashboardsOutput{
				DashboardEntries: []*cloudwatch.DashboardEntry{
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestCloudWatchDashboard_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()
	cw := CloudWatchDashboards{
		Client: mockedCloudWatchDashboard{
			DeleteDashboardsOutput: cloudwatch.DeleteDashboardsOutput{},
		}}

	err := cw.nukeAll([]*string{aws.String("test-name1"), aws.String("test-name2")})
	require.NoError(t, err)
}

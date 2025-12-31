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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockCloudWatchDashboardsClient struct {
	ListDashboardsOutput   cloudwatch.ListDashboardsOutput
	DeleteDashboardsOutput cloudwatch.DeleteDashboardsOutput
}

func (m *mockCloudWatchDashboardsClient) ListDashboards(ctx context.Context, params *cloudwatch.ListDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListDashboardsOutput, error) {
	return &m.ListDashboardsOutput, nil
}

func (m *mockCloudWatchDashboardsClient) DeleteDashboards(ctx context.Context, params *cloudwatch.DeleteDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteDashboardsOutput, error) {
	return &m.DeleteDashboardsOutput, nil
}

func TestListCloudWatchDashboards(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockCloudWatchDashboardsClient{
		ListDashboardsOutput: cloudwatch.ListDashboardsOutput{
			DashboardEntries: []types.DashboardEntry{
				{DashboardName: aws.String("dashboard1"), LastModified: aws.Time(now)},
				{DashboardName: aws.String("dashboard2"), LastModified: aws.Time(now)},
			},
		},
	}

	names, err := listCloudWatchDashboards(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"dashboard1", "dashboard2"}, aws.ToStringSlice(names))
}

func TestListCloudWatchDashboards_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockCloudWatchDashboardsClient{
		ListDashboardsOutput: cloudwatch.ListDashboardsOutput{
			DashboardEntries: []types.DashboardEntry{
				{DashboardName: aws.String("dashboard1"), LastModified: aws.Time(now)},
				{DashboardName: aws.String("skip-this"), LastModified: aws.Time(now)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listCloudWatchDashboards(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"dashboard1"}, aws.ToStringSlice(names))
}

func TestDeleteCloudWatchDashboards(t *testing.T) {
	t.Parallel()

	mock := &mockCloudWatchDashboardsClient{}
	err := deleteCloudWatchDashboards(context.Background(), mock, []string{"test-dashboard"})
	require.NoError(t, err)
}

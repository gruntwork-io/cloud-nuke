package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/grafana"
	"github.com/aws/aws-sdk-go-v2/service/grafana/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockGrafanaClient struct {
	ListWorkspacesOutput  grafana.ListWorkspacesOutput
	DeleteWorkspaceOutput grafana.DeleteWorkspaceOutput
}

func (m *mockGrafanaClient) ListWorkspaces(ctx context.Context, params *grafana.ListWorkspacesInput, optFns ...func(*grafana.Options)) (*grafana.ListWorkspacesOutput, error) {
	return &m.ListWorkspacesOutput, nil
}

func (m *mockGrafanaClient) DeleteWorkspace(ctx context.Context, params *grafana.DeleteWorkspaceInput, optFns ...func(*grafana.Options)) (*grafana.DeleteWorkspaceOutput, error) {
	return &m.DeleteWorkspaceOutput, nil
}

func TestListGrafanaWorkspaces(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := map[string]struct {
		workspaces []types.WorkspaceSummary
		configObj  config.ResourceType
		expected   []string
	}{
		"emptyFilter": {
			workspaces: []types.WorkspaceSummary{
				{Id: aws.String("ws1"), Name: aws.String("ws1"), Created: aws.Time(now), Status: types.WorkspaceStatusActive},
				{Id: aws.String("ws2"), Name: aws.String("ws2"), Created: aws.Time(now), Status: types.WorkspaceStatusActive},
			},
			configObj: config.ResourceType{},
			expected:  []string{"ws1", "ws2"},
		},
		"nameExclusionFilter": {
			workspaces: []types.WorkspaceSummary{
				{Id: aws.String("ws1"), Name: aws.String("ws1"), Created: aws.Time(now), Status: types.WorkspaceStatusActive},
				{Id: aws.String("skip-this"), Name: aws.String("skip-this"), Created: aws.Time(now), Status: types.WorkspaceStatusActive},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
				},
			},
			expected: []string{"ws1"},
		},
		"skipsInactiveStatus": {
			workspaces: []types.WorkspaceSummary{
				{Id: aws.String("active"), Name: aws.String("active"), Created: aws.Time(now), Status: types.WorkspaceStatusActive},
				{Id: aws.String("creating"), Name: aws.String("creating"), Created: aws.Time(now), Status: types.WorkspaceStatusCreating},
			},
			configObj: config.ResourceType{},
			expected:  []string{"active"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockGrafanaClient{ListWorkspacesOutput: grafana.ListWorkspacesOutput{Workspaces: tc.workspaces}}
			ids, err := listGrafanaWorkspaces(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteGrafanaWorkspace(t *testing.T) {
	t.Parallel()
	err := deleteGrafanaWorkspace(context.Background(), &mockGrafanaClient{}, aws.String("test"))
	require.NoError(t, err)
}

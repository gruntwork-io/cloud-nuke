package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/aws/aws-sdk-go-v2/service/amp/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedManagedPrometheusService struct {
	ManagedPrometheusAPI
	ListWorkspacesOutput  amp.ListWorkspacesOutput
	DeleteWorkspaceOutput amp.DeleteWorkspaceOutput
}

func (m mockedManagedPrometheusService) ListWorkspaces(ctx context.Context, input *amp.ListWorkspacesInput, f ...func(*amp.Options)) (*amp.ListWorkspacesOutput, error) {
	return &m.ListWorkspacesOutput, nil
}

func (m mockedManagedPrometheusService) DeleteWorkspace(ctx context.Context, params *amp.DeleteWorkspaceInput, optFns ...func(*amp.Options)) (*amp.DeleteWorkspaceOutput, error) {
	return &m.DeleteWorkspaceOutput, nil
}

func Test_ManagedPrometheus_List(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		workspaces []types.WorkspaceSummary
		configObj  config.ResourceType
		expected   []string
	}{
		"emptyFilter": {
			workspaces: []types.WorkspaceSummary{
				{Alias: aws.String("ws-1"), CreatedAt: &now, Status: &types.WorkspaceStatus{StatusCode: types.WorkspaceStatusCodeActive}, WorkspaceId: aws.String("ws-1")},
				{Alias: aws.String("ws-2"), CreatedAt: aws.Time(now.Add(time.Hour)), Status: &types.WorkspaceStatus{StatusCode: types.WorkspaceStatusCodeActive}, WorkspaceId: aws.String("ws-2")},
			},
			configObj: config.ResourceType{},
			expected:  []string{"ws-1", "ws-2"},
		},
		"nameExclusionFilter": {
			workspaces: []types.WorkspaceSummary{
				{Alias: aws.String("ws-1"), CreatedAt: &now, Status: &types.WorkspaceStatus{StatusCode: types.WorkspaceStatusCodeActive}, WorkspaceId: aws.String("ws-1")},
				{Alias: aws.String("ws-2"), CreatedAt: &now, Status: &types.WorkspaceStatus{StatusCode: types.WorkspaceStatusCodeActive}, WorkspaceId: aws.String("ws-2")},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("ws-1")}},
				},
			},
			expected: []string{"ws-2"},
		},
		"timeAfterExclusionFilter": {
			workspaces: []types.WorkspaceSummary{
				{Alias: aws.String("ws-1"), CreatedAt: &now, Status: &types.WorkspaceStatus{StatusCode: types.WorkspaceStatusCodeActive}, WorkspaceId: aws.String("ws-1")},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
		"skipsNonActiveWorkspaces": {
			workspaces: []types.WorkspaceSummary{
				{Alias: aws.String("active"), CreatedAt: &now, Status: &types.WorkspaceStatus{StatusCode: types.WorkspaceStatusCodeActive}, WorkspaceId: aws.String("active")},
				{Alias: aws.String("creating"), CreatedAt: &now, Status: &types.WorkspaceStatus{StatusCode: types.WorkspaceStatusCodeCreating}, WorkspaceId: aws.String("creating")},
				{Alias: aws.String("deleting"), CreatedAt: &now, Status: &types.WorkspaceStatus{StatusCode: types.WorkspaceStatusCodeDeleting}, WorkspaceId: aws.String("deleting")},
			},
			configObj: config.ResourceType{},
			expected:  []string{"active"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockedManagedPrometheusService{
				ListWorkspacesOutput: amp.ListWorkspacesOutput{Workspaces: tc.workspaces},
			}
			workspaces, err := listManagedPrometheusWorkspaces(context.Background(), client, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, aws.ToStringSlice(workspaces))
		})
	}
}

func Test_ManagedPrometheus_Delete(t *testing.T) {
	t.Parallel()

	client := mockedManagedPrometheusService{
		DeleteWorkspaceOutput: amp.DeleteWorkspaceOutput{},
	}

	workspaceID := "test-workspace-1"
	err := deleteManagedPrometheusWorkspace(context.Background(), client, &workspaceID)
	assert.NoError(t, err)
}

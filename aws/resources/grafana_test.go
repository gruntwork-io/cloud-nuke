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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedGrafanaService struct {
	GrafanaAPI
	DeleteWorkspaceOutput grafana.DeleteWorkspaceOutput
	ListWorkspacesOutput  grafana.ListWorkspacesOutput
}

func (m mockedGrafanaService) DeleteWorkspace(ctx context.Context, params *grafana.DeleteWorkspaceInput, optFns ...func(*grafana.Options)) (*grafana.DeleteWorkspaceOutput, error) {
	return &m.DeleteWorkspaceOutput, nil
}

func (m mockedGrafanaService) ListWorkspaces(ctx context.Context, params *grafana.ListWorkspacesInput, optFns ...func(*grafana.Options)) (*grafana.ListWorkspacesOutput, error) {
	return &m.ListWorkspacesOutput, nil
}

func Test_Grafana_NukeAll(t *testing.T) {
	t.Parallel()

	workspaceName := "test-workspace-1"
	service := Grafana{
		Client: mockedGrafanaService{
			DeleteWorkspaceOutput: grafana.DeleteWorkspaceOutput{},
		},
	}

	err := service.nukeAll([]*string{&workspaceName})
	assert.NoError(t, err)
}

func Test_Grafana_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	workSpace1 := "test-workspace-1"
	workSpace2 := "test-workspace-2"

	service := Grafana{
		Client: mockedGrafanaService{
			ListWorkspacesOutput: grafana.ListWorkspacesOutput{
				Workspaces: []types.WorkspaceSummary{
					{
						Id:      aws.String(workSpace1),
						Name:    aws.String(workSpace1),
						Created: &now,
						Status:  types.WorkspaceStatusActive,
					},
					{
						Id:      aws.String(workSpace2),
						Name:    aws.String(workSpace2),
						Created: aws.Time(now.Add(time.Hour)),
						Status:  types.WorkspaceStatusActive,
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
			expected:  []string{workSpace1, workSpace2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(workSpace1),
					}},
				}},
			expected: []string{workSpace2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			workspaces, err := service.getAll(
				context.Background(),
				config.Config{Grafana: tc.configObj},
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(workspaces))
		})
	}
}

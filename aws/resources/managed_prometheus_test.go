package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/aws/aws-sdk-go-v2/service/amp/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedManagedPrometheusService struct {
	ManagedPrometheusAPI
	ListServiceOutput     amp.ListWorkspacesOutput
	DeleteWorkspaceOutput amp.DeleteWorkspaceOutput
}

func (m mockedManagedPrometheusService) ListWorkspaces(ctx context.Context, input *amp.ListWorkspacesInput, f ...func(*amp.Options)) (*amp.ListWorkspacesOutput, error) {
	return &m.ListServiceOutput, nil
}

func (m mockedManagedPrometheusService) DeleteWorkspace(ctx context.Context, params *amp.DeleteWorkspaceInput, optFns ...func(*amp.Options)) (*amp.DeleteWorkspaceOutput, error) {
	return &m.DeleteWorkspaceOutput, nil
}

func Test_ManagedPrometheus_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()

	workSpace1 := "test-workspace-1"
	workSpace2 := "test-workspace-2"

	service := ManagedPrometheus{
		Client: mockedManagedPrometheusService{
			ListServiceOutput: amp.ListWorkspacesOutput{
				Workspaces: []types.WorkspaceSummary{
					{
						Arn:       aws.String(fmt.Sprintf("arn::%s", workSpace1)),
						Alias:     aws.String(workSpace1),
						CreatedAt: &now,
						Status: &types.WorkspaceStatus{
							StatusCode: types.WorkspaceStatusCodeActive,
						},
						WorkspaceId: aws.String(workSpace1),
					},
					{
						Arn:       aws.String(fmt.Sprintf("arn::%s", workSpace2)),
						Alias:     aws.String(workSpace2),
						CreatedAt: aws.Time(now.Add(time.Hour)),
						Status: &types.WorkspaceStatus{
							StatusCode: types.WorkspaceStatusCodeActive,
						},
						WorkspaceId: aws.String(workSpace2),
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
				config.Config{ManagedPrometheus: tc.configObj},
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(workspaces))
		})
	}
}

func Test_ManagedPrometheus_NukeAll(t *testing.T) {

	t.Parallel()

	workspaceName := "test-workspace-1"
	service := ManagedPrometheus{
		Client: mockedManagedPrometheusService{
			DeleteWorkspaceOutput: amp.DeleteWorkspaceOutput{},
		},
	}

	err := service.nukeAll([]*string{&workspaceName})
	assert.NoError(t, err)
}

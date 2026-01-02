package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/aws/aws-sdk-go-v2/service/amp/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// ManagedPrometheusAPI defines the interface for Managed Prometheus operations.
type ManagedPrometheusAPI interface {
	ListWorkspaces(ctx context.Context, input *amp.ListWorkspacesInput, f ...func(*amp.Options)) (*amp.ListWorkspacesOutput, error)
	DeleteWorkspace(ctx context.Context, params *amp.DeleteWorkspaceInput, optFns ...func(*amp.Options)) (*amp.DeleteWorkspaceOutput, error)
}

// NewManagedPrometheus creates a new ManagedPrometheus resource using the generic resource pattern.
func NewManagedPrometheus() AwsResource {
	return NewAwsResource(&resource.Resource[ManagedPrometheusAPI]{
		ResourceTypeName: "managed-prometheus",
		BatchSize:        100,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ManagedPrometheusAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = amp.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ManagedPrometheus
		},
		Lister: listManagedPrometheusWorkspaces,
		Nuker:  resource.SimpleBatchDeleter(deleteManagedPrometheusWorkspace),
	})
}

// listManagedPrometheusWorkspaces retrieves all Managed Prometheus Workspaces that match the config filters.
func listManagedPrometheusWorkspaces(ctx context.Context, client ManagedPrometheusAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var workspaceIDs []*string

	paginator := amp.NewListWorkspacesPaginator(client, &amp.ListWorkspacesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, workspace := range page.Workspaces {
			if workspace.Status.StatusCode != types.WorkspaceStatusCodeActive {
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: workspace.Alias,
				Time: workspace.CreatedAt,
				Tags: workspace.Tags,
			}) {
				workspaceIDs = append(workspaceIDs, workspace.WorkspaceId)
			}
		}
	}

	return workspaceIDs, nil
}

// deleteManagedPrometheusWorkspace deletes a single Managed Prometheus Workspace.
func deleteManagedPrometheusWorkspace(ctx context.Context, client ManagedPrometheusAPI, workspaceID *string) error {
	_, err := client.DeleteWorkspace(ctx, &amp.DeleteWorkspaceInput{
		WorkspaceId: workspaceID,
	})
	return err
}

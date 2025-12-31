package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/grafana"
	"github.com/aws/aws-sdk-go-v2/service/grafana/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// GrafanaAPI defines the interface for Grafana operations.
type GrafanaAPI interface {
	ListWorkspaces(ctx context.Context, params *grafana.ListWorkspacesInput, optFns ...func(*grafana.Options)) (*grafana.ListWorkspacesOutput, error)
	DeleteWorkspace(ctx context.Context, params *grafana.DeleteWorkspaceInput, optFns ...func(*grafana.Options)) (*grafana.DeleteWorkspaceOutput, error)
}

// NewGrafana creates a new Grafana resource using the generic resource pattern.
func NewGrafana() AwsResource {
	return NewAwsResource(&resource.Resource[GrafanaAPI]{
		ResourceTypeName: "grafana",
		BatchSize:        100,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[GrafanaAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = grafana.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.Grafana
		},
		Lister: listGrafanaWorkspaces,
		Nuker:  resource.SimpleBatchDeleter(deleteGrafanaWorkspace),
	})
}

// listGrafanaWorkspaces retrieves all Grafana Workspaces that match the config filters.
func listGrafanaWorkspaces(ctx context.Context, client GrafanaAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var workspaceIDs []*string

	paginator := grafana.NewListWorkspacesPaginator(client, &grafana.ListWorkspacesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, workspace := range page.Workspaces {
			if workspace.Status != types.WorkspaceStatusActive {
				logging.Debugf(
					"[Grafana] skipping grafana workspace: %s, status: %s",
					*workspace.Name,
					workspace.Status,
				)

				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: workspace.Name,
				Time: workspace.Created,
				Tags: workspace.Tags,
			}) {
				workspaceIDs = append(workspaceIDs, workspace.Id)
			}
		}
	}

	return workspaceIDs, nil
}

// deleteGrafanaWorkspace deletes a single Grafana Workspace.
func deleteGrafanaWorkspace(ctx context.Context, client GrafanaAPI, workspaceID *string) error {
	_, err := client.DeleteWorkspace(ctx, &grafana.DeleteWorkspaceInput{
		WorkspaceId: workspaceID,
	})
	return err
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/grafana"
	"github.com/aws/aws-sdk-go-v2/service/grafana/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (g *Grafana) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	var workspaceIDs []*string

	paginator := grafana.NewListWorkspacesPaginator(g.Client, &grafana.ListWorkspacesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, workspace := range page.Workspaces {
			if workspace.Status != types.WorkspaceStatusActive {
				logging.Debugf(
					"[Grafana] skiping grafana workspaces: %s, status: %s",
					*workspace.Name,
					workspace.Status,
				)

				continue
			}

			if cnfObj.Grafana.ShouldInclude(config.ResourceValue{
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

func (g *Grafana) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Grafana] No Grafana Workspaces found in region %s", g.Region)
		return nil
	}

	logging.Debugf("[Grafana] Deleting all Grafana Workspaces in %s", g.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		_, err := g.Client.DeleteWorkspace(g.Context, &grafana.DeleteWorkspaceInput{
			WorkspaceId: identifier,
		})
		if err != nil {
			logging.Debugf("[Grafana] Error deleting Workspace %s in region %s", *identifier, g.Region)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[Grafana] Deleted Workspace %s in region %s", *identifier, g.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: g.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Grafana Workspace(s) deleted in %s", len(deleted), g.Region)
	return nil
}

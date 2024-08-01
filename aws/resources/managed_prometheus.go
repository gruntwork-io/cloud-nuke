package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/aws/aws-sdk-go-v2/service/amp/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (a *ManagedPrometheus) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Managed Prometheus] No Prometheus Workspaces found in region %s", a.Region)
		return nil
	}

	logging.Debugf("[Managed Prometheus] Deleting all Prometheus Workspaces in %s", a.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		logging.Debugf("[Managed Prometheus] Deleting Prometheus Workspace %s in region %s", *identifier, a.Region)

		_, err := a.Client.DeleteWorkspace(a.Context, &amp.DeleteWorkspaceInput{
			WorkspaceId: identifier,
			ClientToken: nil,
		})
		if err != nil {
			logging.Debugf("[Managed Prometheus] Error deleting Workspace %s in region %s", *identifier, a.Region)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[Managed Prometheus] Deleted Workspace %s in region %s", *identifier, a.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: a.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Prometheus Workspace(s) deleted in %s", len(deleted), a.Region)
	return nil
}

func (a *ManagedPrometheus) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	paginator := amp.NewListWorkspacesPaginator(a.Client, &amp.ListWorkspacesInput{})

	var identifiers []*string
	for paginator.HasMorePages() {
		workspaces, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Managed Prometheus] Failed to list workspaces: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, workspace := range workspaces.Workspaces {
			if workspace.Status.StatusCode != types.WorkspaceStatusCodeActive {
				continue
			}

			if cnfObj.ManagedPrometheus.ShouldInclude(config.ResourceValue{
				Name: workspace.Alias,
				Time: workspace.CreatedAt,
				Tags: workspace.Tags,
			}) {
				identifiers = append(identifiers, workspace.WorkspaceId)
			}
		}
	}

	return identifiers, nil
}

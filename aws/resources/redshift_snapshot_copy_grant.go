package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (g *RedshiftSnapshotCopyGrants) getAll(ctx context.Context, configObj config.Config) ([]*string, error) {
	var grantNames []*string
	var marker *string

	for {
		output, err := g.Client.DescribeSnapshotCopyGrants(ctx, &redshift.DescribeSnapshotCopyGrantsInput{
			Marker: marker,
		})
		if err != nil {
			logging.Debugf("[Redshift Snapshot Copy Grant] Failed to list grants: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, grant := range output.SnapshotCopyGrants {
			if configObj.RedshiftSnapshotCopyGrant.ShouldInclude(config.ResourceValue{
				Name: grant.SnapshotCopyGrantName,
			}) {
				grantNames = append(grantNames, grant.SnapshotCopyGrantName)
			}
		}

		if output.Marker == nil {
			break
		}
		marker = output.Marker
	}

	return grantNames, nil
}

func (g *RedshiftSnapshotCopyGrants) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Redshift Snapshot Copy Grants to nuke in region %s", g.Region)
		return nil
	}

	logging.Debugf("Deleting all Redshift Snapshot Copy Grants in region %s", g.Region)

	deletedCount := 0
	for _, name := range identifiers {
		_, err := g.Client.DeleteSnapshotCopyGrant(g.Context, &redshift.DeleteSnapshotCopyGrantInput{
			SnapshotCopyGrantName: name,
		})

		e := report.Entry{
			Identifier:   aws.ToString(name),
			ResourceType: "Redshift Snapshot Copy Grant",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Errorf("[Failed] %s: %s", aws.ToString(name), err)
		} else {
			deletedCount++
			logging.Debugf("Deleted Redshift Snapshot Copy Grant: %s", aws.ToString(name))
		}
	}

	logging.Debugf("[OK] %d Redshift Snapshot Copy Grant(s) deleted in %s", deletedCount, g.Region)
	return nil
}

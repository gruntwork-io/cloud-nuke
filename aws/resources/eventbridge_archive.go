package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (eba *EventBridgeArchive) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	var identifiers []*string

	params := eventbridge.ListArchivesInput{}
	hasMorePages := true

	for hasMorePages {
		archives, err := eba.Client.ListArchives(ctx, &params)
		if err != nil {
			logging.Debugf("[Event Bridge Archives] Failed to list archives: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, archive := range archives.Archives {
			if cnfObj.EventBridgeArchive.ShouldInclude(config.ResourceValue{
				Name: archive.ArchiveName,
				Time: archive.CreationTime,
			}) {
				identifiers = append(identifiers, archive.ArchiveName)
			}
		}

		params.NextToken = archives.NextToken
		hasMorePages = params.NextToken != nil
	}

	return identifiers, nil
}

func (eba *EventBridgeArchive) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Event Bridge Archive] No Archives found in region %s", eba.Region)
		return nil
	}

	logging.Debugf("[Event Bridge Archive] Deleting all Archives in %s", eba.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		_, err := eba.Client.DeleteArchive(context.Background(), &eventbridge.DeleteArchiveInput{
			ArchiveName: identifier,
		})
		if err != nil {
			logging.Debugf(
				"[Event Bridge Archive] Error deleting Archive %s in region %s, err %s",
				*identifier,
				eba.Region,
				err,
			)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[Event Bridge Archive] Deleted Archive %s in region %s", *identifier, eba.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: eba.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Event Bridge Archive(s) deleted in %s", len(deleted), eba.Region)
	return nil
}

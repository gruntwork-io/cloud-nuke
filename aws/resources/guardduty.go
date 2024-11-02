package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

type DetectorOutputWithID struct {
	ID     *string
	Output *guardduty.GetDetectorOutput
}

func (gd *GuardDuty) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var detectorIdsToInclude []*string
	paginator := guardduty.NewListDetectorsPaginator(gd.Client, &guardduty.ListDetectorsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Due to the ListDetectors method only returning the Ids of found detectors, we need to further enrich our data about
		// each detector with a separate call to GetDetector for metadata including when it was created, which we need to make the
		// determination about whether or not the given detector should be included
		for _, detectorId := range page.DetectorIds {
			detector, err := gd.Client.GetDetector(gd.Context, &guardduty.GetDetectorInput{
				DetectorId: aws.String(detectorId),
			})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if gd.shouldInclude(detector, aws.String(detectorId), configObj) {
				detectorIdsToInclude = append(detectorIdsToInclude, aws.String(detectorId))
			}
		}
	}

	return detectorIdsToInclude, nil
}

func (gd *GuardDuty) shouldInclude(detector *guardduty.GetDetectorOutput, detectorId *string, configObj config.Config) bool {
	createdAtDateTime, err := util.ParseTimestamp(detector.CreatedAt)
	if err != nil {
		logging.Debugf("Could not parse createdAt timestamp (%s) of GuardDuty detector %s. Excluding from delete.", *createdAtDateTime, *detectorId)
	}

	return configObj.GuardDuty.ShouldInclude(config.ResourceValue{Time: createdAtDateTime})
}

func (gd *GuardDuty) nukeAll(detectorIds []string) error {
	if len(detectorIds) == 0 {
		logging.Debugf("No GuardDuty detectors to nuke in region %s", gd.Region)

		return nil
	}

	logging.Debugf("Deleting all GuardDuty detectors in region %s", gd.Region)

	var deletedIds []string

	for _, detectorId := range detectorIds {
		params := &guardduty.DeleteDetectorInput{
			DetectorId: aws.String(detectorId),
		}

		_, err := gd.Client.DeleteDetector(gd.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   detectorId,
			ResourceType: "GuardDuty Detector",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s: %s", detectorId, err)
		} else {
			deletedIds = append(deletedIds, detectorId)
			logging.Debugf("Deleted GuardDuty detector: %s", detectorId)
		}
	}

	logging.Debugf("[OK] %d GuardDuty Detector(s) deleted in %s", len(deletedIds), gd.Region)
	return nil
}

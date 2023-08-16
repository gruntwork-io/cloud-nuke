package resources

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

type DetectorOutputWithID struct {
	ID     *string
	Output *guardduty.GetDetectorOutput
}

func (gd *GuardDuty) getAll(configObj config.Config) ([]*string, error) {
	var detectorIdsToInclude []*string
	var detectorIds []*string
	err := gd.Client.ListDetectorsPages(&guardduty.ListDetectorsInput{}, func(page *guardduty.ListDetectorsOutput, lastPage bool) bool {
		detectorIds = append(detectorIds, page.DetectorIds...)
		return !lastPage
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// Due to the ListDetectors method only returning the Ids of found detectors, we need to further enrich our data about
	// each detector with a separate call to GetDetector for metadata including when it was created, which we need to make the
	// determination about whether or not the given detector should be included
	for _, detectorId := range detectorIds {
		detector, err := gd.Client.GetDetector(&guardduty.GetDetectorInput{
			DetectorId: detectorId,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if gd.shouldInclude(detector, detectorId, configObj) {
			detectorIdsToInclude = append(detectorIdsToInclude, detectorId)
		}
	}

	return detectorIdsToInclude, nil
}

func (gd *GuardDuty) shouldInclude(detector *guardduty.GetDetectorOutput, detectorId *string, configObj config.Config) bool {
	detectorCreatedAt := aws.StringValue(detector.CreatedAt)
	createdAtDateTime, err := time.Parse(time.RFC3339, detectorCreatedAt)
	if err != nil {
		logging.Logger.Debugf("Could not parse createdAt timestamp (%s) of GuardDuty detector %s. Excluding from delete.", detectorCreatedAt, *detectorId)
	}

	return configObj.GuardDuty.ShouldInclude(config.ResourceValue{Time: &createdAtDateTime})
}

func (gd *GuardDuty) nukeAll(detectorIds []string) error {
	if len(detectorIds) == 0 {
		logging.Logger.Debugf("No GuardDuty detectors to nuke in region %s", gd.Region)

		return nil
	}

	logging.Logger.Debugf("Deleting all GuardDuty detectors in region %s", gd.Region)

	deletedIds := []string{}

	for _, detectorId := range detectorIds {
		params := &guardduty.DeleteDetectorInput{
			DetectorId: aws.String(detectorId),
		}

		_, err := gd.Client.DeleteDetector(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   detectorId,
			ResourceType: "GuardDuty Detector",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s: %s", detectorId, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking GuardDuty Detector",
			}, map[string]interface{}{
				"region": gd.Region,
			})
		} else {
			deletedIds = append(deletedIds, detectorId)
			logging.Logger.Debugf("Deleted GuardDuty detector: %s", detectorId)
		}
	}

	logging.Logger.Debugf("[OK] %d GuardDuty Detector(s) deleted in %s", len(deletedIds), gd.Region)
	return nil
}

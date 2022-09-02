package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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

func getAllGuardDutyDetectors(session *session.Session, excludeAfter time.Time, configObj config.Config, batchSize int) ([]string, error) {
	svc := guardduty.New(session)

	var result []*string
	var annotatedDetectors []*DetectorOutputWithID
	var detectorIdsToInclude []string

	var next *string = nil
	for {
		list, err := svc.ListDetectors(&guardduty.ListDetectorsInput{
			MaxResults: awsgo.Int64(int64(batchSize)),
			NextToken:  next,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		result = append(result, list.DetectorIds...)
		if list.NextToken == nil || len(list.DetectorIds) == 0 {
			break
		}
		next = list.NextToken
	}

	// Due to the ListDetectors method only returning the Ids of found detectors, we need to further enrich our data about
	// each detector with a separate call to GetDetector for metadata including when it was created, which we need to make the
	// determination about whether or not the given detector should be included
	for _, detectorId := range result {

		detector, getDetectorErr := svc.GetDetector(&guardduty.GetDetectorInput{
			DetectorId: detectorId,
		})

		if getDetectorErr != nil {
			return nil, errors.WithStackTrace(getDetectorErr)
		}

		detectorOutputWithID := &DetectorOutputWithID{
			ID:     detectorId,
			Output: detector,
		}

		annotatedDetectors = append(annotatedDetectors, detectorOutputWithID)
	}

	for _, detector := range annotatedDetectors {
		if shouldIncludeDetector(detector, excludeAfter, configObj) {
			detectorIdsToInclude = append(detectorIdsToInclude, aws.StringValue(detector.ID))
		}
	}

	return detectorIdsToInclude, nil
}

func shouldIncludeDetector(detector *DetectorOutputWithID, excludeAfter time.Time, configObj config.Config) bool {
	if detector == nil {
		return false
	}

	detectorCreatedAt := aws.StringValue(detector.Output.CreatedAt)

	createdAtDateTime, err := time.Parse(time.RFC3339, detectorCreatedAt)
	if err != nil {
		logging.Logger.Warnf("Could not parse createdAt timestamp (%s) of GuardDuty detector %s. Excluding from delete.", detectorCreatedAt, awsgo.StringValue(detector.ID))
	}

	if excludeAfter.Before(createdAtDateTime) {
		return false
	}

	return true
}

func nukeAllGuardDutyDetectors(session *session.Session, detectorIds []string) error {
	svc := guardduty.New(session)

	if len(detectorIds) == 0 {
		logging.Logger.Infof("No GuardDuty detectors to nuke in region %s", *session.Config.Region)

		return nil
	}

	logging.Logger.Infof("Deleting all GuardDuty detectors in region %s", *session.Config.Region)

	deletedIds := []string{}

	for _, detectorId := range detectorIds {
		params := &guardduty.DeleteDetectorInput{
			DetectorId: aws.String(detectorId),
		}

		_, err := svc.DeleteDetector(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   detectorId,
			ResourceType: "GuardDuty Detector",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", detectorId, err)
		} else {
			deletedIds = append(deletedIds, detectorId)
			logging.Logger.Infof("Deleted GuardDuty detector: %s", detectorId)
		}
	}

	logging.Logger.Infof("[OK] %d GuardDuty Detector(s) deleted in %s", len(deletedIds), *session.Config.Region)

	return nil
}

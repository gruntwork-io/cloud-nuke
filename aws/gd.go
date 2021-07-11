package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAllGuardDutyDetectors returns active GuardDuty detectors in a given region.
func getAllGuardDutyDetectors(session *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := guardduty.New(session)

	ctx := context.Background()
	var detectors []*string

	err := svc.ListDetectorsPagesWithContext(ctx, &guardduty.ListDetectorsInput{},
		func(d *guardduty.ListDetectorsOutput, lastPage bool) bool {
			for _, detector := range d.DetectorIds {
				detectors = append(detectors, detector)
			}
			return true
		})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var results []*string

	for _, detector := range detectors {
		layout := "2006-01-02T15:04:05.000Z"
		output, err := svc.GetDetector(&guardduty.GetDetectorInput{DetectorId: detector})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		createdTime, err := time.Parse(layout, *output.CreatedAt)
		if excludeAfter.After(createdTime) {
			results = append(results, detector)
		}

	}

	return results, nil
}

// nukeAllGuardDutyDetectors disables GuardDuty for an AWS region. This will also remove all associated findings.
func nukeAllGuardDutyDetectors(session *session.Session, detectors []*string) error {
	svc := guardduty.New(session)

	if len(detectors) == 0 {
		logging.Logger.Infof("GuardDuty was not found to be enabled in an region: %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all GuardDuty instances in region: %s", *session.Config.Region)

	ctx := context.Background()
	for _, detector := range detectors {
		_, err := svc.DeleteDetectorWithContext(ctx, &guardduty.DeleteDetectorInput{
			DetectorId: detector,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			logging.Logger.Infof("Deleted GuardDuty instance: %s", *detector)
		}
	}

	return nil
}

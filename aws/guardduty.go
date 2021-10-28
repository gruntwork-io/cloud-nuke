package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// Returns a formatted string of ELB names
func getAllGuardDutyDetectors(session *session.Session, region string) ([]*string, error) {
	svc := guardduty.New(session)
	result, err := svc.ListDetectors(&guardduty.ListDetectorsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return result.DetectorIds, nil
}

func deleteDetector(svc *guardduty.GuardDuty, detectorId *string) error {
	_, err := svc.DeleteDetector(&guardduty.DeleteDetectorInput{
		DetectorId: detectorId,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Nuke a single user
func nukeDetector(svc *guardduty.GuardDuty, userName *string) error {
	functions := []func(svc *guardduty.GuardDuty, detectorId *string) error{
		deleteDetector,
	}

	for _, fn := range functions {
		if err := fn(svc, userName); err != nil {
			return err
		}
	}

	return nil
}

// Delete all GuardDuty Detectors
func nukeAllGuardDutyDetectors(session *session.Session, detectors []*string) error {
	if len(detectors) == 0 {
		logging.Logger.Info("No GuardDuty Detectors to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all GuardDuty Detectors")

	deletedUsers := 0
	svc := guardduty.New(session)
	multiErr := new(multierror.Error)

	for _, detector := range detectors {
		err := nukeDetector(svc, detector)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		} else {
			deletedUsers++
			logging.Logger.Infof("Deleted GuardDuty Detectors: %s", *detector)
		}
	}

	logging.Logger.Infof("[OK] %d GuardDuty Detector(s) terminated", deletedUsers)
	return multiErr.ErrorOrNil()
}

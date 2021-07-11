package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
)

// createTestGuardDuty creates a GuardDuty detector
func createTestGuardDuty(t *testing.T, session *session.Session) string {
	svc := guardduty.New(session)

	detector, err := svc.CreateDetector(&guardduty.CreateDetectorInput{
		Enable: awsgo.Bool(true),
	})
	if err != nil {
		assert.Failf(t, "Could not create test GuardDuty instance", errors.WithStackTrace(err).Error())
	}

	return *detector.DetectorId
}

func TestNukeAllGuardDutyDetectors(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	sess, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	detectorID := createTestGuardDuty(t, sess)
	defer func(session *session.Session, detectors []*string) {
		err := nukeAllGuardDutyDetectors(session, detectors)
		if err != nil {
			assert.Fail(t, errors.WithStackTrace(err).Error())
		}
	}(sess, []*string{&detectorID})

	detectors, err := getAllGuardDutyDetectors(sess, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to list GuardDuty detector in region")
	}

	assert.NotContains(t, awsgo.StringValueSlice(detectors), detectorID)

	detectors, err = getAllGuardDutyDetectors(sess, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to list GuardDuty detector in region")
	}

	assert.Contains(t, awsgo.StringValueSlice(detectors), detectorID)
}

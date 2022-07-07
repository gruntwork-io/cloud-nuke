package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
)

type GuardDuty struct {
	detectorIds []string
}

func (gd GuardDuty) ResourceName() string {
	return "guardduty"
}

func (gd GuardDuty) ResourceIdentifiers() []string {
	return gd.detectorIds
}

func (gd GuardDuty) MaxBatchSize() int {
	return 10
}

func (gd GuardDuty) Nuke(session *session.Session, detectorIds []string) error {
	return nukeAllGuardDutyDetectors(session, detectorIds)
}

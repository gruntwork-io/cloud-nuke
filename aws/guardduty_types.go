package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/guardduty/guarddutyiface"
)

type GuardDuty struct {
	Client      guarddutyiface.GuardDutyAPI
	Region      string
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
	return gd.nukeAll(detectorIds)
}

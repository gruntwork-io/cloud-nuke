package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/aws/aws-sdk-go/service/guardduty/guarddutyiface"
	"github.com/gruntwork-io/cloud-nuke/config"
)

type GuardDuty struct {
	Client      guarddutyiface.GuardDutyAPI
	Region      string
	detectorIds []string
}

func (gd *GuardDuty) Init(session *session.Session) {
	gd.Client = guardduty.New(session)
}

func (gd *GuardDuty) ResourceName() string {
	return "guardduty"
}

func (gd *GuardDuty) ResourceIdentifiers() []string {
	return gd.detectorIds
}

func (gd *GuardDuty) MaxBatchSize() int {
	return 10
}

func (gd *GuardDuty) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := gd.getAll(configObj)
	if err != nil {
		return nil, err
	}

	gd.detectorIds = awsgo.StringValueSlice(identifiers)
	return gd.detectorIds, nil
}

func (gd *GuardDuty) Nuke(detectorIds []string) error {
	return gd.nukeAll(detectorIds)
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/config"
)

type GuardDutyAPI interface {
	GetDetector(ctx context.Context, params *guardduty.GetDetectorInput, optFns ...func(*guardduty.Options)) (*guardduty.GetDetectorOutput, error)
	DeleteDetector(ctx context.Context, params *guardduty.DeleteDetectorInput, optFns ...func(*guardduty.Options)) (*guardduty.DeleteDetectorOutput, error)
	ListDetectors(ctx context.Context, params *guardduty.ListDetectorsInput, optFns ...func(*guardduty.Options)) (*guardduty.ListDetectorsOutput, error)
}

type GuardDuty struct {
	BaseAwsResource
	Client      GuardDutyAPI
	Region      string
	detectorIds []string
}

func (gd *GuardDuty) InitV2(cfg aws.Config) {
	gd.Client = guardduty.NewFromConfig(cfg)
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

func (gd *GuardDuty) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.GuardDuty
}

func (gd *GuardDuty) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := gd.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	gd.detectorIds = aws.ToStringSlice(identifiers)
	return gd.detectorIds, nil
}

func (gd *GuardDuty) Nuke(detectorIds []string) error {
	return gd.nukeAll(detectorIds)
}

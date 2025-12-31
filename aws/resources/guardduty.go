package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// GuardDutyAPI defines the interface for GuardDuty operations.
type GuardDutyAPI interface {
	GetDetector(ctx context.Context, params *guardduty.GetDetectorInput, optFns ...func(*guardduty.Options)) (*guardduty.GetDetectorOutput, error)
	DeleteDetector(ctx context.Context, params *guardduty.DeleteDetectorInput, optFns ...func(*guardduty.Options)) (*guardduty.DeleteDetectorOutput, error)
	ListDetectors(ctx context.Context, params *guardduty.ListDetectorsInput, optFns ...func(*guardduty.Options)) (*guardduty.ListDetectorsOutput, error)
}

// NewGuardDuty creates a new GuardDuty resource using the generic resource pattern.
func NewGuardDuty() AwsResource {
	return NewAwsResource(&resource.Resource[GuardDutyAPI]{
		ResourceTypeName: "guardduty",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[GuardDutyAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = guardduty.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GuardDuty
		},
		Lister: listGuardDutyDetectors,
		Nuker:  resource.SimpleBatchDeleter(deleteGuardDutyDetector),
	})
}

// listGuardDutyDetectors retrieves all GuardDuty detectors that match the config filters.
func listGuardDutyDetectors(ctx context.Context, client GuardDutyAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var detectorIds []*string

	paginator := guardduty.NewListDetectorsPaginator(client, &guardduty.ListDetectorsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, detectorId := range page.DetectorIds {
			detector, err := client.GetDetector(ctx, &guardduty.GetDetectorInput{
				DetectorId: aws.String(detectorId),
			})
			if err != nil {
				return nil, err
			}

			createdAt, err := util.ParseTimestamp(detector.CreatedAt)
			if err != nil {
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{Time: createdAt}) {
				detectorIds = append(detectorIds, aws.String(detectorId))
			}
		}
	}

	return detectorIds, nil
}

// deleteGuardDutyDetector deletes a single GuardDuty detector.
func deleteGuardDutyDetector(ctx context.Context, client GuardDutyAPI, detectorId *string) error {
	_, err := client.DeleteDetector(ctx, &guardduty.DeleteDetectorInput{
		DetectorId: detectorId,
	})
	return err
}

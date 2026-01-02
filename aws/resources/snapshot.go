package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// SnapshotsAPI defines the interface for EBS Snapshot operations.
type SnapshotsAPI interface {
	DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error)
	DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error)
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
	DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error)
}

// NewSnapshots creates a new Snapshots resource using the generic resource pattern.
func NewSnapshots() AwsResource {
	return NewAwsResource(&resource.Resource[SnapshotsAPI]{
		ResourceTypeName: "snap",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SnapshotsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.Snapshots
		},
		Lister: listSnapshots,
		Nuker:  resource.MultiStepDeleter(deregisterSnapshotAMIs, deleteSnapshot),
		PermissionVerifier: func(ctx context.Context, client SnapshotsAPI, id *string) error {
			_, err := client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
				SnapshotId: id,
				DryRun:     aws.Bool(true),
			})
			return err
		},
	})
}

// listSnapshots retrieves all EBS Snapshots that match the config filters.
func listSnapshots(ctx context.Context, client SnapshotsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// status - The status of the snapshot (pending | completed | error).
	// We only want to list EBS Snapshots with a status of "completed" or "error"
	// since those are the only statuses eligible for deletion.
	statusFilter := types.Filter{Name: aws.String("status"), Values: []string{"completed", "error"}}

	var snapshotIds []*string
	paginator := ec2.NewDescribeSnapshotsPaginator(client, &ec2.DescribeSnapshotsInput{
		OwnerIds: []string{"self"},
		Filters:  []types.Filter{statusFilter},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, snapshot := range page.Snapshots {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: snapshot.StartTime,
				Tags: util.ConvertTypesTagsToMap(snapshot.Tags),
			}) && !snapshotHasAWSBackupTag(snapshot.Tags) {
				snapshotIds = append(snapshotIds, snapshot.SnapshotId)
			}
		}
	}

	return snapshotIds, nil
}

// snapshotHasAWSBackupTag checks if the snapshot has an AWS Backup tag.
// Resources created by AWS Backup are listed as owned by self, but are actually
// AWS managed resources and cannot be deleted here.
func snapshotHasAWSBackupTag(tags []types.Tag) bool {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "aws:backup:source-resource" {
			return true
		}
	}
	return false
}

// deregisterSnapshotAMIs deregisters any AMIs that were created from the snapshot.
func deregisterSnapshotAMIs(ctx context.Context, client SnapshotsAPI, snapshotID *string) error {
	logging.Debugf("De-registering images for snapshot: %s", aws.ToString(snapshotID))

	output, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("block-device-mapping.snapshot-id"),
				Values: []string{aws.ToString(snapshotID)},
			},
		},
	})
	if err != nil {
		logging.Debugf("[Describe Images] Failed to describe images for snapshot: %s", aws.ToString(snapshotID))
		return err
	}

	for _, image := range output.Images {
		_, err := client.DeregisterImage(ctx, &ec2.DeregisterImageInput{
			ImageId: image.ImageId,
		})
		if err != nil {
			logging.Debugf("[Failed] de-registering image %v for snapshot: %s", aws.ToString(image.ImageId), aws.ToString(snapshotID))
			return err
		}
	}

	logging.Debugf("[Ok] De-registered all the images for snapshot: %s", aws.ToString(snapshotID))
	return nil
}

// deleteSnapshot deletes a single EBS Snapshot.
func deleteSnapshot(ctx context.Context, client SnapshotsAPI, snapshotID *string) error {
	_, err := client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
		SnapshotId: snapshotID,
	})
	return err
}

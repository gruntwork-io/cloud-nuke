package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// AMIsAPI defines the interface for AMI operations.
type AMIsAPI interface {
	DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error)
	DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error)
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

// NewAMIs creates a new AMIs resource using the generic resource pattern.
func NewAMIs() AwsResource {
	return NewAwsResource(&resource.Resource[AMIsAPI]{
		ResourceTypeName: "ami",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[AMIsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.AMI
		},
		Lister: listAMIs,
		Nuker:  resource.SimpleBatchDeleter(nukeAMI),
	})
}

// listAMIs retrieves all user-owned AMIs that match the config filters.
func listAMIs(ctx context.Context, client AMIsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var imageIds []*string
	paginator := ec2.NewDescribeImagesPaginator(client, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, image := range page.Images {
			if shouldSkipAMI(image) {
				continue
			}

			createdTime, err := util.ParseTimestamp(image.CreationDate)
			if err != nil {
				return nil, err
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: image.Name,
				Time: createdTime,
				Tags: util.ConvertTypesTagsToMap(image.Tags),
			}) {
				imageIds = append(imageIds, image.ImageId)
			}
		}
	}

	return imageIds, nil
}

// shouldSkipAMI checks if an AMI should be skipped (AWS managed or AWS Backup).
func shouldSkipAMI(image types.Image) bool {
	// Skip images created by AWS Backup
	if image.Name != nil && strings.HasPrefix(*image.Name, "AwsBackup") {
		return true
	}

	// Check if the image has a tag that indicates AWS management
	for _, tag := range image.Tags {
		if aws.ToString(tag.Key) == "aws-managed" && aws.ToString(tag.Value) == "true" {
			return true
		}
	}

	return false
}

// nukeAMI deregisters an AMI and deletes its associated EBS snapshots.
// Note: Snapshots that are shared with other AMIs may fail to delete.
func nukeAMI(ctx context.Context, client AMIsAPI, imageID *string) error {
	// First, get the AMI details to find associated snapshots
	snapshotIDs, err := getAMISnapshotIDs(ctx, client, imageID)
	if err != nil {
		logging.Debugf("Failed to get snapshot IDs for AMI %s: %v", aws.ToString(imageID), err)
		// Continue with deregistration even if we can't get snapshots
	}

	// Deregister the AMI
	if _, err := client.DeregisterImage(ctx, &ec2.DeregisterImageInput{
		ImageId: imageID,
	}); err != nil {
		return err
	}

	// Delete associated snapshots (best effort - some may be shared)
	for _, snapshotID := range snapshotIDs {
		if _, err := client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
			SnapshotId: snapshotID,
		}); err != nil {
			// Log but don't fail - snapshot may be used by other AMIs
			logging.Debugf("Failed to delete snapshot %s for AMI %s: %v",
				aws.ToString(snapshotID), aws.ToString(imageID), err)
		}
	}

	return nil
}

// getAMISnapshotIDs returns the EBS snapshot IDs associated with an AMI.
func getAMISnapshotIDs(ctx context.Context, client AMIsAPI, imageID *string) ([]*string, error) {
	output, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{aws.ToString(imageID)},
	})
	if err != nil {
		return nil, err
	}

	if len(output.Images) == 0 {
		return nil, nil
	}

	var snapshotIDs []*string
	for _, mapping := range output.Images[0].BlockDeviceMappings {
		if mapping.Ebs != nil && mapping.Ebs.SnapshotId != nil {
			snapshotIDs = append(snapshotIDs, mapping.Ebs.SnapshotId)
		}
	}

	return snapshotIDs, nil
}

// ImageAvailableError is returned when an image doesn't become available within wait attempts.
type ImageAvailableError struct{}

func (e ImageAvailableError) Error() string {
	return "Image didn't become available within wait attempts"
}

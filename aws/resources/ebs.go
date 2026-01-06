package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// EBSVolumesAPI defines the interface for EBS Volume operations.
type EBSVolumesAPI interface {
	DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
	DeleteVolume(ctx context.Context, params *ec2.DeleteVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVolumeOutput, error)
}

// NewEBSVolumes creates a new EBS Volumes resource using the generic resource pattern.
func NewEBSVolumes() AwsResource {
	return NewAwsResource(&resource.Resource[EBSVolumesAPI]{
		ResourceTypeName: "ebs",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EBSVolumesAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EBSVolume
		},
		Lister:             listEBSVolumes,
		Nuker:              resource.SequentialDeleteThenWaitAll(deleteEBSVolume, waitForEBSVolumesDeleted),
		PermissionVerifier: verifyEBSVolumePermission,
	})
}

// listEBSVolumes retrieves all EBS volumes that match the config filters.
// Only lists volumes in deletable states: available, creating, or error.
func listEBSVolumes(ctx context.Context, client EBSVolumesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var volumeIds []*string

	// Only list volumes eligible for deletion (not in-use or deleting)
	statusFilter := types.Filter{
		Name:   aws.String("status"),
		Values: []string{"available", "creating", "error"},
	}

	paginator := ec2.NewDescribeVolumesPaginator(client, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{statusFilter},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, volume := range page.Volumes {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: util.GetEC2ResourceNameTagValue(volume.Tags),
				Time: volume.CreateTime,
				Tags: util.ConvertTypesTagsToMap(volume.Tags),
			}) {
				volumeIds = append(volumeIds, volume.VolumeId)
			}
		}
	}

	return volumeIds, nil
}

// verifyEBSVolumePermission performs a dry-run delete to check permissions.
func verifyEBSVolumePermission(ctx context.Context, client EBSVolumesAPI, id *string) error {
	_, err := client.DeleteVolume(ctx, &ec2.DeleteVolumeInput{
		VolumeId: id,
		DryRun:   aws.Bool(true),
	})
	return err
}

// deleteEBSVolume deletes a single EBS volume.
func deleteEBSVolume(ctx context.Context, client EBSVolumesAPI, volumeID *string) error {
	_, err := client.DeleteVolume(ctx, &ec2.DeleteVolumeInput{
		VolumeId: volumeID,
	})
	return err
}

// waitForEBSVolumesDeleted waits for all specified EBS volumes to be fully deleted.
func waitForEBSVolumesDeleted(ctx context.Context, client EBSVolumesAPI, ids []string) error {
	waiter := ec2.NewVolumeDeletedWaiter(client)
	return waiter.Wait(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: ids,
	}, 5*time.Minute)
}

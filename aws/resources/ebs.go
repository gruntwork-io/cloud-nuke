package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
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
		BatchSize:        49,
		InitClient: func(r *resource.Resource[EBSVolumesAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EC2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ec2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EBSVolume
		},
		Lister:             listEBSVolumes,
		Nuker:              resource.SequentialDeleteThenWaitAll(deleteEBSVolume, waitForEBSVolumesDeleted),
		PermissionVerifier: verifyEBSVolumePermission,
	})
}

// listEBSVolumes retrieves all EBS volumes that match the config filters.
func listEBSVolumes(ctx context.Context, client EBSVolumesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// Available statuses: (creating | available | in-use | deleting | deleted | error).
	// Since the output of this function is used to delete the returned volumes
	// We want to only list EBS volumes with a status of "available" or "creating"
	// Since those are the only statuses that are eligible for deletion
	statusFilter := types.Filter{Name: aws.String("status"), Values: []string{"available", "creating", "error"}}
	result, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{statusFilter},
	})

	if err != nil {
		return nil, err
	}

	var volumeIds []*string
	for _, volume := range result.Volumes {
		if shouldIncludeEBSVolume(volume, cfg) {
			volumeIds = append(volumeIds, volume.VolumeId)
		}
	}

	return volumeIds, nil
}

func shouldIncludeEBSVolume(volume types.Volume, cfg config.ResourceType) bool {
	name := ""
	for _, tag := range volume.Tags {
		if aws.ToString(tag.Key) == "Name" {
			name = aws.ToString(tag.Value)
		}
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: &name,
		Time: volume.CreateTime,
		Tags: util.ConvertTypesTagsToMap(volume.Tags),
	})
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

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// ElasticFileSystemAPI defines the interface for EFS operations.
type ElasticFileSystemAPI interface {
	DeleteAccessPoint(ctx context.Context, params *efs.DeleteAccessPointInput, optFns ...func(*efs.Options)) (*efs.DeleteAccessPointOutput, error)
	DeleteFileSystem(ctx context.Context, params *efs.DeleteFileSystemInput, optFns ...func(*efs.Options)) (*efs.DeleteFileSystemOutput, error)
	DeleteMountTarget(ctx context.Context, params *efs.DeleteMountTargetInput, optFns ...func(*efs.Options)) (*efs.DeleteMountTargetOutput, error)
	DescribeAccessPoints(ctx context.Context, params *efs.DescribeAccessPointsInput, optFns ...func(*efs.Options)) (*efs.DescribeAccessPointsOutput, error)
	DescribeMountTargets(ctx context.Context, params *efs.DescribeMountTargetsInput, optFns ...func(*efs.Options)) (*efs.DescribeMountTargetsOutput, error)
	DescribeFileSystems(ctx context.Context, params *efs.DescribeFileSystemsInput, optFns ...func(*efs.Options)) (*efs.DescribeFileSystemsOutput, error)
}

// NewElasticFileSystem creates a new ElasticFileSystem resource using the generic resource pattern.
func NewElasticFileSystem() AwsResource {
	return NewAwsResource(&resource.Resource[ElasticFileSystemAPI]{
		ResourceTypeName: "efs",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ElasticFileSystemAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = efs.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ElasticFileSystem
		},
		Lister: listElasticFileSystems,
		// EFS deletion requires sequential steps: access points → mount targets → wait → delete
		Nuker: resource.MultiStepDeleter(
			deleteEFSAccessPoints,
			deleteEFSMountTargets,
			waitForEFSMountTargetsDeleted,
			deleteEFSFileSystem,
		),
	})
}

// listElasticFileSystems retrieves all Elastic File Systems that match the config filters.
func listElasticFileSystems(ctx context.Context, client ElasticFileSystemAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allEfs []*string

	paginator := efs.NewDescribeFileSystemsPaginator(client, &efs.DescribeFileSystemsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, system := range page.FileSystems {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: system.Name,
				Time: system.CreationTime,
			}) {
				allEfs = append(allEfs, system.FileSystemId)
			}
		}
	}

	return allEfs, nil
}

// deleteEFSAccessPoints deletes all access points for the given EFS.
// EFS cannot be deleted while access points exist.
func deleteEFSAccessPoints(ctx context.Context, client ElasticFileSystemAPI, efsID *string) error {
	paginator := efs.NewDescribeAccessPointsPaginator(client, &efs.DescribeAccessPointsInput{
		FileSystemId: efsID,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, ap := range page.AccessPoints {
			logging.Debugf("Deleting access point %s for EFS %s", aws.ToString(ap.AccessPointId), aws.ToString(efsID))
			if _, err := client.DeleteAccessPoint(ctx, &efs.DeleteAccessPointInput{
				AccessPointId: ap.AccessPointId,
			}); err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}

	return nil
}

// deleteEFSMountTargets deletes all mount targets for the given EFS.
// EFS cannot be deleted while mount targets exist.
func deleteEFSMountTargets(ctx context.Context, client ElasticFileSystemAPI, efsID *string) error {
	paginator := efs.NewDescribeMountTargetsPaginator(client, &efs.DescribeMountTargetsInput{
		FileSystemId: efsID,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, mt := range page.MountTargets {
			logging.Debugf("Deleting mount target %s for EFS %s", aws.ToString(mt.MountTargetId), aws.ToString(efsID))
			if _, err := client.DeleteMountTarget(ctx, &efs.DeleteMountTargetInput{
				MountTargetId: mt.MountTargetId,
			}); err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}

	return nil
}

// waitForEFSMountTargetsDeleted polls until all mount targets for the given EFS are deleted.
// Times out after 60 seconds.
func waitForEFSMountTargetsDeleted(ctx context.Context, client ElasticFileSystemAPI, efsID *string) error {
	for i := 0; i < 30; i++ {
		output, err := client.DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
			FileSystemId: efsID,
		})
		if err != nil {
			// Error (like FileSystemNotFound) means mount targets are gone
			return nil //nolint:nilerr
		}
		if len(output.MountTargets) == 0 {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out waiting for mount targets to be deleted for EFS %s", aws.ToString(efsID))
}

// deleteEFSFileSystem deletes the EFS file system.
func deleteEFSFileSystem(ctx context.Context, client ElasticFileSystemAPI, efsID *string) error {
	_, err := client.DeleteFileSystem(ctx, &efs.DeleteFileSystemInput{
		FileSystemId: efsID,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
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
		InitClient: func(r *resource.Resource[ElasticFileSystemAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EFS client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = efs.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ElasticFileSystem
		},
		Lister: listElasticFileSystems,
		Nuker:  deleteElasticFileSystems,
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

// deleteElasticFileSystems is a custom nuker for Elastic File Systems.
// It deletes access points, mount targets, and then the file system.
func deleteElasticFileSystems(ctx context.Context, client ElasticFileSystemAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Elastic FileSystems (efs) to nuke in %s", scope)
		return nil
	}

	if len(identifiers) > 100 {
		logging.Debugf("Nuking too many Elastic FileSystems (100): halting to avoid hitting AWS API rate limiting")
		return TooManyElasticFileSystemsErr{}
	}

	logging.Debugf("Deleting Elastic FileSystems (efs) in %s", scope)

	var allErrs *multierror.Error
	for _, efsID := range identifiers {
		if err := deleteElasticFileSystem(ctx, client, scope, resourceType, efsID); err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Debugf("[Failed] %s", err)
		}
	}

	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func deleteElasticFileSystem(ctx context.Context, client ElasticFileSystemAPI, scope resource.Scope, resourceType string, efsID *string) error {
	var allErrs *multierror.Error

	// First, we need to check if the Elastic FileSystem is "in-use", because an in-use file system cannot be deleted
	// An Elastic FileSystem is considered in-use if it has any access points, or any mount targets
	// Here, we first look up and delete any and all access points for the given Elastic FileSystem
	var accessPointIds []*string

	accessPointParam := &efs.DescribeAccessPointsInput{
		FileSystemId: efsID,
	}

	out, err := client.DescribeAccessPoints(ctx, accessPointParam)
	if err != nil {
		allErrs = multierror.Append(allErrs, err)
	}

	for _, ap := range out.AccessPoints {
		accessPointIds = append(accessPointIds, ap.AccessPointId)
	}

	// Delete all access points in a loop
	for _, apID := range accessPointIds {
		deleteParam := &efs.DeleteAccessPointInput{
			AccessPointId: apID,
		}

		logging.Debugf("Deleting access point (id=%s) for Elastic FileSystem (%s) in %s", aws.ToString(apID), aws.ToString(efsID), scope)

		_, err := client.DeleteAccessPoint(ctx, deleteParam)
		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		} else {
			logging.Debugf("[OK] Deleted access point (id=%s) for Elastic FileSystem (%s) in %s", aws.ToString(apID), aws.ToString(efsID), scope)
		}
	}

	// With Access points cleared up, we next turn to looking up and deleting mount targets
	// Note that, despite having a MaxItems field in its struct, DescribeMountTargetsInput will actually
	// only set this value to 10, ignoring any other values. This means we must page through our mount targets,
	// because we must guarantee they are all deleted before we can successfully delete the Elastic FileSystem itself
	done := false
	var marker *string

	var mountTargetIds []*string

	for !done {

		mountTargetParam := &efs.DescribeMountTargetsInput{
			FileSystemId: efsID,
		}

		// If the last iteration had a marker set, use it
		if aws.ToString(marker) != "" {
			mountTargetParam.Marker = marker
		}

		mountTargetsOutput, describeMountsErr := client.DescribeMountTargets(ctx, mountTargetParam)
		if describeMountsErr != nil {
			allErrs = multierror.Append(allErrs, describeMountsErr)
		}

		for _, mountTarget := range mountTargetsOutput.MountTargets {
			mountTargetIds = append(mountTargetIds, mountTarget.MountTargetId)
		}

		// If the response contained a NextMarker field, set it as the next iteration's marker
		if aws.ToString(mountTargetsOutput.NextMarker) != "" {
			marker = mountTargetsOutput.NextMarker
		} else {
			// There's no NextMarker set on the response, so we're done enumerating mount targets
			done = true
		}
	}

	for _, mtID := range mountTargetIds {
		deleteMtParam := &efs.DeleteMountTargetInput{
			MountTargetId: mtID,
		}

		logging.Debugf("Deleting mount target (id=%s) for Elastic FileSystem (%s) in %s", aws.ToString(mtID), aws.ToString(efsID), scope)

		_, err := client.DeleteMountTarget(ctx, deleteMtParam)
		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		} else {
			logging.Debugf("[OK] Deleted mount target (id=%s) for Elastic FileSystem (%s) in %s", aws.ToString(mtID), aws.ToString(efsID), scope)
		}
	}

	logging.Debug("Sleeping 20 seconds to allow AWS to realize the Elastic FileSystem is no longer in use...")
	time.Sleep(20 * time.Second)

	// Now we can attempt to delete the Elastic FileSystem itself
	deleteEfsParam := &efs.DeleteFileSystemInput{
		FileSystemId: efsID,
	}

	_, deleteErr := client.DeleteFileSystem(ctx, deleteEfsParam)
	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.ToString(efsID),
		ResourceType: resourceType,
		Error:        deleteErr,
	}
	report.Record(e)

	if deleteErr != nil {
		allErrs = multierror.Append(allErrs, deleteErr)
	}

	if deleteErr == nil {
		logging.Debugf("[OK] Elastic FileSystem (efs) %s deleted in %s", aws.ToString(efsID), scope)
	} else {
		logging.Debugf("[Failed] Error deleting Elastic FileSystem (efs) %s in %s", aws.ToString(efsID), scope)
	}

	return allErrs.ErrorOrNil()
}

// custom errors

type TooManyElasticFileSystemsErr struct{}

func (err TooManyElasticFileSystemsErr) Error() string {
	return "Too many Elastic FileSystems requested at once."
}

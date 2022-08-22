package aws

import (
	"context"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllElasticFileSystems(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}
	svc := efs.NewFromConfig(cfg)

	result, err := svc.DescribeFileSystems(context.TODO(), &efs.DescribeFileSystemsInput{})
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}

	Ids := []*string{}
	for _, fileSystem := range result.FileSystems {
		if shouldIncludeElasticFileSystem(&fileSystem, excludeAfter, configObj) {
			Ids = append(Ids, fileSystem.FileSystemId)
		}
	}
	return Ids, nil
}

func shouldIncludeElasticFileSystem(efsDescription *types.FileSystemDescription, excludeAfter time.Time, configObj config.Config) bool {
	if efsDescription == nil {
		return false
	}

	if efsDescription.CreationTime != nil {
		if excludeAfter.Before(aws.TimeValue(efsDescription.CreationTime)) {
			return false
		}
	}

	return config.ShouldInclude(
		aws.StringValue(efsDescription.Name),
		configObj.ElasticFileSystem.IncludeRule.NamesRegExp,
		configObj.ElasticFileSystem.ExcludeRule.NamesRegExp,
	)
}

func nukeAllElasticFileSystems(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return errors.WithStackTrace(err)
	}
	svc := efs.NewFromConfig(cfg)

	if len(identifiers) == 0 {
		logging.Logger.Infof("No Elastic FileSystems (efs) to nuke in region %s", region)
	}

	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many Elastic FileSystems (100): halting to avoid hitting AWS API rate limiting")
		return TooManyElasticFileSystemsErr{}
	}

	// There is no bulk delete EFS API, so we delete the batch of Elastic FileSystems concurrently using goroutines
	logging.Logger.Infof("Deleting Elastic FileSystems (efs) in region %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, efsID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteElasticFileSystemAsync(wg, errChans[i], svc, efsID, region)
	}
	wg.Wait()

	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func deleteElasticFileSystemAsync(wg *sync.WaitGroup, errChan chan error, svc *efs.Client, efsID *string, region string) {
	var allErrs *multierror.Error

	defer wg.Done()
	defer func() { errChan <- allErrs.ErrorOrNil() }()

	// First, we need to check if the Elastic FileSystem is in use. It is considered in-use if it has any mount points,
	// so we first describe and delete any existing mount points
	accessPointIds := []*string{}

	accessPointParam := &efs.DescribeAccessPointsInput{
		FileSystemId: efsID,
	}

	out, err := svc.DescribeAccessPoints(context.TODO(), accessPointParam)
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

		logging.Logger.Infof("Deleting access point (id=%s) for Elastic FileSystem (%s) in region: %s", aws.StringValue(apID), aws.StringValue(efsID), region)

		_, err := svc.DeleteAccessPoint(context.TODO(), deleteParam)
		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		} else {
			logging.Logger.Infof("[OK] Deleted access point (id=%s) for Elastic FileSystem (%s) in region: %s", aws.StringValue(apID), aws.StringValue(efsID), region)
		}
	}

	// Until we've paginated through all of the Elastic FileSysem's mount targets, we have to keep going
	done := false
	var marker *string

	mountTargetIds := []*string{}

	for !done {

		// We also need to ensure the Elastic FileSystem does not have any mount points
		mountTargetParam := &efs.DescribeMountTargetsInput{
			FileSystemId: efsID,
		}

		// If the last iteration had a marker set, use it
		if aws.StringValue(marker) != "" {
			mountTargetParam.Marker = marker
		}

		mountTargetsOutput, describeMountsErr := svc.DescribeMountTargets(context.TODO(), mountTargetParam)
		if describeMountsErr != nil {
			allErrs = multierror.Append(allErrs, err)
		}

		for _, mountTarget := range mountTargetsOutput.MountTargets {
			mountTargetIds = append(mountTargetIds, mountTarget.MountTargetId)
		}

		// If the response contained a NextMarker field, set it as the next iteration's marker
		if aws.StringValue(mountTargetsOutput.NextMarker) != "" {
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

		logging.Logger.Infof("Deleting mount target (id=%s) for Elastic FileSystem (%s) in region: %s", aws.StringValue(mtID), aws.StringValue(efsID), region)

		_, err := svc.DeleteMountTarget(context.TODO(), deleteMtParam)
		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		} else {
			logging.Logger.Infof("[OK] Deleted mount target (id=%s) for Elastic FileSystem (%s) in region: %s", aws.StringValue(mtID), aws.StringValue(efsID), region)
		}
	}

	logging.Logger.Info("Sleeping 20 seconds to allow AWS to realize the Elastic FileSystem is no longer in use...")
	time.Sleep(20 * time.Second)

	// Now we can attempt to delete the Elastic FileSystem itself
	deleteEfsParam := &efs.DeleteFileSystemInput{
		FileSystemId: efsID,
	}

	_, deleteErr := svc.DeleteFileSystem(context.TODO(), deleteEfsParam)
	if deleteErr != nil {
		allErrs = multierror.Append(allErrs, deleteErr)
	}

	if err == nil {
		logging.Logger.Infof("[OK] Elastic FileSystem (efs) %s deleted in %s", aws.StringValue(efsID), region)
	} else {
		logging.Logger.Errorf("[Failed] Error deleting Elastic FileSystem (efs) %s in %s", aws.StringValue(efsID), region)
	}
}

package resources

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (ef *ElasticFileSystem) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var allEfs []*string

	paginator := efs.NewDescribeFileSystemsPaginator(ef.Client, &efs.DescribeFileSystemsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, system := range page.FileSystems {
			if configObj.ElasticFileSystem.ShouldInclude(config.ResourceValue{
				Name: system.Name,
				Time: system.CreationTime,
			}) {
				allEfs = append(allEfs, system.FileSystemId)
			}
		}
	}

	return allEfs, nil
}

func (ef *ElasticFileSystem) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Elastic FileSystems (efs) to nuke in region %s", ef.Region)
	}

	if len(identifiers) > 100 {
		logging.Debugf("Nuking too many Elastic FileSystems (100): halting to avoid hitting AWS API rate limiting")
		return TooManyElasticFileSystemsErr{}
	}

	// There is no bulk delete EFS API, so we delete the batch of Elastic FileSystems concurrently using goroutines
	logging.Debugf("Deleting Elastic FileSystems (efs) in region %s", ef.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, efsID := range identifiers {
		errChans[i] = make(chan error, 1)
		go ef.deleteAsync(wg, errChans[i], efsID)
	}
	wg.Wait()

	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
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

func (ef *ElasticFileSystem) deleteAsync(wg *sync.WaitGroup, errChan chan error, efsID *string) {
	var allErrs *multierror.Error

	defer wg.Done()
	defer func() { errChan <- allErrs.ErrorOrNil() }()

	// First, we need to check if the Elastic FileSystem is "in-use", because an in-use file system cannot be deleted
	// An Elastic FileSystem is considered in-use if it has any access points, or any mount targets
	// Here, we first look up and delete any and all access points for the given Elastic FileSystem
	var accessPointIds []*string

	accessPointParam := &efs.DescribeAccessPointsInput{
		FileSystemId: efsID,
	}

	out, err := ef.Client.DescribeAccessPoints(ef.Context, accessPointParam)
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

		logging.Debugf("Deleting access point (id=%s) for Elastic FileSystem (%s) in region: %s", aws.ToString(apID), aws.ToString(efsID), ef.Region)

		_, err := ef.Client.DeleteAccessPoint(ef.Context, deleteParam)
		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		} else {
			logging.Debugf("[OK] Deleted access point (id=%s) for Elastic FileSystem (%s) in region: %s", aws.ToString(apID), aws.ToString(efsID), ef.Region)
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

		mountTargetsOutput, describeMountsErr := ef.Client.DescribeMountTargets(ef.Context, mountTargetParam)
		if describeMountsErr != nil {
			allErrs = multierror.Append(allErrs, err)
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

		logging.Debugf("Deleting mount target (id=%s) for Elastic FileSystem (%s) in region: %s", aws.ToString(mtID), aws.ToString(efsID), ef.Region)

		_, err := ef.Client.DeleteMountTarget(ef.Context, deleteMtParam)
		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		} else {
			logging.Debugf("[OK] Deleted mount target (id=%s) for Elastic FileSystem (%s) in region: %s", aws.ToString(mtID), aws.ToString(efsID), ef.Region)
		}
	}

	logging.Debug("Sleeping 20 seconds to allow AWS to realize the Elastic FileSystem is no longer in use...")
	time.Sleep(20 * time.Second)

	// Now we can attempt to delete the Elastic FileSystem itself
	deleteEfsParam := &efs.DeleteFileSystemInput{
		FileSystemId: efsID,
	}

	_, deleteErr := ef.Client.DeleteFileSystem(ef.Context, deleteEfsParam)
	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.ToString(efsID),
		ResourceType: "Elastic FileSystem (EFS)",
		Error:        err,
	}
	report.Record(e)

	if deleteErr != nil {
		allErrs = multierror.Append(allErrs, deleteErr)
	}

	if err == nil {
		logging.Debugf("[OK] Elastic FileSystem (efs) %s deleted in %s", aws.ToString(efsID), ef.Region)
	} else {
		logging.Debugf("[Failed] Error deleting Elastic FileSystem (efs) %s in %s", aws.ToString(efsID), ef.Region)
	}
}

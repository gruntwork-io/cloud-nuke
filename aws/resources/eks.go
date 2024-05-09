package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// getAll returns a list of strings of EKS Cluster Names that uniquely identify each cluster.
func (clusters *EKSClusters) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := clusters.Client.ListClustersWithContext(clusters.Context, &eks.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	filteredClusters, err := clusters.filter(result.Clusters, configObj)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return filteredClusters, nil
}

// filter will take in the list of clusters and filter out any clusters that were created after
// `excludeAfter`, and those that are excluded by the config file.
func (clusters *EKSClusters) filter(clusterNames []*string, configObj config.Config) ([]*string, error) {
	var filteredEksClusterNames []*string
	for _, clusterName := range clusterNames {
		// Since we already have the name here, avoid an extra API call by applying
		// the name based config filter first.
		if !configObj.EKSCluster.ShouldInclude(config.ResourceValue{Name: clusterName}) {
			continue
		}

		describeResult, err := clusters.Client.DescribeClusterWithContext(
			clusters.Context,
			&eks.DescribeClusterInput{Name: clusterName})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if !configObj.EKSCluster.ShouldInclude(config.ResourceValue{
			Time: describeResult.Cluster.CreatedAt,
			Tags: util.ConvertStringPtrTagsToMap(describeResult.Cluster.Tags),
		}) {
			continue
		}

		filteredEksClusterNames = append(filteredEksClusterNames, clusterName)
	}

	return filteredEksClusterNames, nil
}

// deleteAsync deletes the provided EKS Cluster asynchronously in a goroutine, using wait groups for
// concurrency control and a return channel for errors. Note that this routine attempts to delete all managed compute
// resources associated with the EKS cluster (Managed Node Groups and Fargate Profiles).
func (clusters *EKSClusters) deleteAsync(wg *sync.WaitGroup, errChan chan error, eksClusterName string) {
	defer wg.Done()

	// Aggregate errors for each subresource being deleted
	var allSubResourceErrs error

	// Since deleting node groups can take some time, we first schedule the deletion of them, and then move on to
	// deleting the Fargate profiles so that they can be done in parallel, before waiting for the node groups to be
	// deleted.
	deletedNodeGroups, err := clusters.scheduleDeleteEKSClusterManagedNodeGroup(eksClusterName)
	if err != nil {
		allSubResourceErrs = multierror.Append(allSubResourceErrs, err)
	}

	if err := clusters.deleteEKSClusterFargateProfiles(eksClusterName); err != nil {
		allSubResourceErrs = multierror.Append(allSubResourceErrs, err)
	}

	// Make sure the node groups are actually deleted before returning.
	for _, nodeGroup := range deletedNodeGroups {
		err := clusters.Client.WaitUntilNodegroupDeletedWithContext(clusters.Context, &eks.DescribeNodegroupInput{
			ClusterName:   aws.String(eksClusterName),
			NodegroupName: nodeGroup,
		})
		if err != nil {
			logging.Debugf("[Failed] Failed waiting for Node Group %s associated with cluster %s to be deleted: %s", aws.StringValue(nodeGroup), eksClusterName, err)
			allSubResourceErrs = multierror.Append(allSubResourceErrs, err)
		} else {
			logging.Debugf("Deleted Node Group %s associated with cluster %s", aws.StringValue(nodeGroup), eksClusterName)
		}
	}
	if allSubResourceErrs != nil {
		errChan <- allSubResourceErrs
		return
	}

	// At this point, all the sub resources of the EKS cluster has been confirmed to be deleted, so we can go ahead to
	// request to delete the EKS cluster.
	_, deleteErr := clusters.Client.DeleteClusterWithContext(clusters.Context, &eks.DeleteClusterInput{Name: aws.String(eksClusterName)})
	if deleteErr != nil {
		logging.Debugf("[Failed] Failed deleting EKS cluster %s: %s", eksClusterName, deleteErr)
	}
	errChan <- deleteErr
}

// scheduleDeleteEKSClusterManagedNodeGroup looks up all the associated Managed Node Group resources on the EKS cluster
// and requests each one to be deleted. Note that this function will not wait for the Node Groups to be deleted. This
// will return the list of Node Groups that were successfully scheduled for deletion.
func (clusters *EKSClusters) scheduleDeleteEKSClusterManagedNodeGroup(eksClusterName string) ([]*string, error) {
	allNodeGroups := []*string{}
	err := clusters.Client.ListNodegroupsPagesWithContext(
		clusters.Context,
		&eks.ListNodegroupsInput{ClusterName: aws.String(eksClusterName)},
		func(page *eks.ListNodegroupsOutput, lastPage bool) bool {
			allNodeGroups = append(allNodeGroups, page.Nodegroups...)
			return !lastPage
		},
	)
	if err != nil {
		return nil, err
	}

	// Since there isn't a bulk node group delete, we make the requests to delete node groups in a tight loop. This
	// doesn't implement pagination or throttling because the assumption is that the EKS Clusters being deleted by
	// cloud-nuke should be fairly small due to its use case. We can improve this if anyone runs into scalability
	// issues with this implementation.
	var allDeleteErrs error
	deletedNodeGroups := []*string{}
	for _, nodeGroup := range allNodeGroups {
		_, err := clusters.Client.DeleteNodegroupWithContext(
			clusters.Context,
			&eks.DeleteNodegroupInput{
				ClusterName:   aws.String(eksClusterName),
				NodegroupName: nodeGroup,
			})
		if err != nil {
			logging.Debugf("[Failed] Failed deleting Node Group %s associated with cluster %s: %s", aws.StringValue(nodeGroup), eksClusterName, err)
			allDeleteErrs = multierror.Append(allDeleteErrs, err)
		} else {
			deletedNodeGroups = append(deletedNodeGroups, nodeGroup)
		}
	}
	return deletedNodeGroups, allDeleteErrs
}

// deleteEKSClusterFargateProfiles looks up all the associated Fargate Profile resources on the EKS cluster and requests
// each one to be deleted. Since only one Fargate Profile can be deleted at a time, this function will wait until the
// Fargate Profile is actually deleted for each one before moving on to the next one.
func (clusters *EKSClusters) deleteEKSClusterFargateProfiles(eksClusterName string) error {
	allFargateProfiles := []*string{}
	err := clusters.Client.ListFargateProfilesPagesWithContext(
		clusters.Context,
		&eks.ListFargateProfilesInput{ClusterName: aws.String(eksClusterName)},
		func(page *eks.ListFargateProfilesOutput, lastPage bool) bool {
			allFargateProfiles = append(allFargateProfiles, page.FargateProfileNames...)
			return !lastPage
		},
	)
	if err != nil {
		return err
	}

	// We can't delete Fargate profiles in parallel, so we delete them sequentially, waiting until the profile has been
	// deleted before moving on to the next one. This will make the delete routine very slow, but unfortunately, there
	// is no other alternative.
	// See https://docs.aws.amazon.com/eks/latest/APIReference/API_DeleteFargateProfile.html for more info on the serial
	// requirement.
	// Note that we aggregate deletion errors so that we at least attempt to delete all of them once.
	var allDeleteErrs error
	for _, fargateProfile := range allFargateProfiles {
		_, err := clusters.Client.DeleteFargateProfileWithContext(
			clusters.Context,
			&eks.DeleteFargateProfileInput{
				ClusterName:        aws.String(eksClusterName),
				FargateProfileName: fargateProfile,
			})
		if err != nil {
			logging.Debugf("[Failed] Failed deleting Fargate Profile %s associated with cluster %s: %s", aws.StringValue(fargateProfile), eksClusterName, err)
			allDeleteErrs = multierror.Append(allDeleteErrs, err)
			continue
		}

		waitErr := clusters.Client.WaitUntilFargateProfileDeletedWithContext(
			clusters.Context,
			&eks.DescribeFargateProfileInput{
				ClusterName:        aws.String(eksClusterName),
				FargateProfileName: fargateProfile,
			})
		if waitErr != nil {
			logging.Debugf("[Failed] Failed waiting for Fargate Profile %s associated with cluster %s to be deleted: %s", aws.StringValue(fargateProfile), eksClusterName, waitErr)
			allDeleteErrs = multierror.Append(allDeleteErrs, waitErr)
		} else {
			logging.Debugf("Deleted Fargate Profile %s associated with cluster %s", aws.StringValue(fargateProfile), eksClusterName)
		}
	}
	return allDeleteErrs
}

// waitUntilEksClustersDeleted waits until the EKS cluster has been actually deleted from AWS. Returns a list of EKS
// cluster names that have been successfully deleted.
func (clusters *EKSClusters) waitUntilEksClustersDeleted(eksClusterNames []*string) []*string {
	var successfullyDeleted []*string
	for _, eksClusterName := range eksClusterNames {
		err := clusters.Client.WaitUntilClusterDeletedWithContext(clusters.Context, &eks.DescribeClusterInput{Name: eksClusterName})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(eksClusterName),
			ResourceType: "EKS Cluster",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] Failed waiting for EKS cluster to be deleted %s: %s", *eksClusterName, err)
		} else {
			logging.Debugf("Deleted EKS cluster: %s", aws.StringValue(eksClusterName))
			successfullyDeleted = append(successfullyDeleted, eksClusterName)
		}
	}
	return successfullyDeleted
}

// nukeAll deletes all provided EKS clusters, waiting for them to be deleted before returning.
func (clusters *EKSClusters) nukeAll(eksClusterNames []*string) error {
	numNuking := len(eksClusterNames)
	if numNuking == 0 {
		logging.Debugf("No EKS clusters to nuke in region %s", clusters.Region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on EKSCluster.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if numNuking > 100 {
		logging.Debugf("Nuking too many EKS Clusters at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyEKSClustersErr{}
	}

	// We need to delete subresources associated with the EKS Cluster before being able to delete the cluster, so we
	// spawn goroutines to drive the deletion of each EKS cluster.
	logging.Debugf("Deleting %d EKS clusters in region %s", numNuking, clusters.Region)
	wg := new(sync.WaitGroup)
	wg.Add(numNuking)
	errChans := make([]chan error, numNuking)
	for i, eksClusterName := range eksClusterNames {
		errChans[i] = make(chan error, 1)
		go clusters.deleteAsync(wg, errChans[i], aws.StringValue(eksClusterName))
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
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

	// Now wait until the EKS Clusters are deleted
	successfullyDeleted := clusters.waitUntilEksClustersDeleted(eksClusterNames)
	numNuked := len(successfullyDeleted)
	logging.Debugf("[OK] %d of %d EKS cluster(s) deleted in %s", numNuked, numNuking, clusters.Region)
	return nil
}

// Custom errors

type TooManyEKSClustersErr struct{}

func (err TooManyEKSClustersErr) Error() string {
	return "Too many EKS Clusters requested at once."
}

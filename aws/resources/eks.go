package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// EKSClustersAPI defines the interface for EKS Clusters operations.
type EKSClustersAPI interface {
	DeleteCluster(ctx context.Context, params *eks.DeleteClusterInput, optFns ...func(*eks.Options)) (*eks.DeleteClusterOutput, error)
	DeleteFargateProfile(ctx context.Context, params *eks.DeleteFargateProfileInput, optFns ...func(*eks.Options)) (*eks.DeleteFargateProfileOutput, error)
	DeleteNodegroup(ctx context.Context, params *eks.DeleteNodegroupInput, optFns ...func(*eks.Options)) (*eks.DeleteNodegroupOutput, error)
	DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
	DescribeFargateProfile(ctx context.Context, params *eks.DescribeFargateProfileInput, optFns ...func(*eks.Options)) (*eks.DescribeFargateProfileOutput, error)
	DescribeNodegroup(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error)
	ListClusters(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error)
	ListFargateProfiles(ctx context.Context, params *eks.ListFargateProfilesInput, optFns ...func(*eks.Options)) (*eks.ListFargateProfilesOutput, error)
	ListNodegroups(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error)
}

// NewEKSClusters creates a new EKS Clusters resource using the generic resource pattern.
func NewEKSClusters() AwsResource {
	return NewAwsResource(&resource.Resource[EKSClustersAPI]{
		ResourceTypeName: "ekscluster",
		// Tentative batch size to ensure AWS doesn't throttle. Note that deleting EKS clusters involves deleting many
		// associated sub resources in tight loops, and they happen in parallel in go routines. We conservatively pick 10
		// here, both to limit overloading the runtime and to avoid AWS throttling with many API calls.
		BatchSize: 10,
		InitClient: func(r *resource.Resource[EKSClustersAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EKS client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = eks.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EKSCluster
		},
		Lister: listEKSClusters,
		Nuker:  deleteEKSClusters,
	})
}

// listEKSClusters retrieves all EKS clusters that match the config filters.
func listEKSClusters(ctx context.Context, client EKSClustersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	filteredClusters, err := filterEKSClusters(ctx, client, aws.StringSlice(result.Clusters), cfg)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return filteredClusters, nil
}

// filterEKSClusters filters EKS clusters based on the config.
func filterEKSClusters(ctx context.Context, client EKSClustersAPI, clusterNames []*string, cfg config.ResourceType) ([]*string, error) {
	var filteredEksClusterNames []*string
	for _, clusterName := range clusterNames {
		describeResult, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: clusterName})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if !cfg.ShouldInclude(config.ResourceValue{
			Name: clusterName,
			Time: describeResult.Cluster.CreatedAt,
			Tags: describeResult.Cluster.Tags,
		}) {
			continue
		}

		filteredEksClusterNames = append(filteredEksClusterNames, clusterName)
	}

	return filteredEksClusterNames, nil
}

// deleteEKSClusters is a custom nuker function for EKS clusters.
// EKS clusters require deleting sub-resources (node groups, fargate profiles) before deletion.
func deleteEKSClusters(ctx context.Context, client EKSClustersAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	numNuking := len(identifiers)
	if numNuking == 0 {
		logging.Debugf("No EKS clusters to nuke in region %s", scope.Region)
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

	// We need to delete sub-resources associated with the EKS Cluster before being able to delete the cluster, so we
	// spawn goroutines to drive the deletion of each EKS cluster.
	logging.Debugf("Deleting %d EKS clusters in region %s", numNuking, scope.Region)
	wg := new(sync.WaitGroup)
	wg.Add(numNuking)
	errChans := make([]chan error, numNuking)
	for i, eksClusterName := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteEKSClusterAsync(ctx, client, wg, errChans[i], aws.ToString(eksClusterName))
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
	successfullyDeleted := waitUntilEksClustersDeleted(ctx, client, resourceType, identifiers)
	numNuked := len(successfullyDeleted)
	logging.Debugf("[OK] %d of %d EKS cluster(s) deleted in %s", numNuked, numNuking, scope.Region)
	return nil
}

// deleteEKSClusterAsync deletes the provided EKS Cluster asynchronously in a goroutine, using wait groups for
// concurrency control and a return channel for errors. Note that this routine attempts to delete all managed compute
// resources associated with the EKS cluster (Managed Node Groups and Fargate Profiles).
func deleteEKSClusterAsync(ctx context.Context, client EKSClustersAPI, wg *sync.WaitGroup, errChan chan error, eksClusterName string) {
	defer wg.Done()

	// Aggregate errors for each subresource being deleted
	var allSubResourceErrs error

	// Since deleting node groups can take some time, we first schedule the deletion of them, and then move on to
	// deleting the Fargate profiles so that they can be done in parallel, before waiting for the node groups to be
	// deleted.
	deletedNodeGroups, err := scheduleDeleteEKSClusterManagedNodeGroup(ctx, client, eksClusterName)
	if err != nil {
		allSubResourceErrs = multierror.Append(allSubResourceErrs, err)
	}

	if err := deleteEKSClusterFargateProfiles(ctx, client, eksClusterName); err != nil {
		allSubResourceErrs = multierror.Append(allSubResourceErrs, err)
	}

	// Make sure the node groups are actually deleted before returning.
	for _, nodeGroup := range deletedNodeGroups {
		waiter := eks.NewNodegroupDeletedWaiter(client)
		err := waiter.Wait(ctx, &eks.DescribeNodegroupInput{
			ClusterName:   aws.String(eksClusterName),
			NodegroupName: nodeGroup,
		}, DefaultWaitTimeout)
		if err != nil {
			logging.Debugf("[Failed] Failed waiting for Node Group %s associated with cluster %s to be deleted: %s", aws.ToString(nodeGroup), eksClusterName, err)
			allSubResourceErrs = multierror.Append(allSubResourceErrs, err)
		} else {
			logging.Debugf("Deleted Node Group %s associated with cluster %s", aws.ToString(nodeGroup), eksClusterName)
		}
	}
	if allSubResourceErrs != nil {
		errChan <- allSubResourceErrs
		return
	}

	// At this point, all the sub resources of the EKS cluster has been confirmed to be deleted, so we can go ahead to
	// request to delete the EKS cluster.
	_, deleteErr := client.DeleteCluster(ctx, &eks.DeleteClusterInput{Name: aws.String(eksClusterName)})
	if deleteErr != nil {
		logging.Debugf("[Failed] Failed deleting EKS cluster %s: %s", eksClusterName, deleteErr)
	}
	errChan <- deleteErr
}

// scheduleDeleteEKSClusterManagedNodeGroup looks up all the associated Managed Node Group resources on the EKS cluster
// and requests each one to be deleted. Note that this function will not wait for the Node Groups to be deleted. This
// will return the list of Node Groups that were successfully scheduled for deletion.
func scheduleDeleteEKSClusterManagedNodeGroup(ctx context.Context, client EKSClustersAPI, eksClusterName string) ([]*string, error) {
	var allNodeGroups []*string

	paginator := eks.NewListNodegroupsPaginator(client, &eks.ListNodegroupsInput{
		ClusterName: aws.String(eksClusterName),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		allNodeGroups = append(allNodeGroups, aws.StringSlice(page.Nodegroups)...)
	}

	// Since there isn't a bulk node group delete, we make the requests to delete node groups in a tight loop. This
	// doesn't implement pagination or throttling because the assumption is that the EKS Clusters being deleted by
	// cloud-nuke should be fairly small due to its use case. We can improve this if anyone runs into scalability
	// issues with this implementation.
	var allDeleteErrs error
	var deletedNodeGroups []*string
	for _, nodeGroup := range allNodeGroups {
		_, err := client.DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
			ClusterName:   aws.String(eksClusterName),
			NodegroupName: nodeGroup,
		})
		if err != nil {
			logging.Debugf("[Failed] Failed deleting Node Group %s associated with cluster %s: %s", aws.ToString(nodeGroup), eksClusterName, err)
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
func deleteEKSClusterFargateProfiles(ctx context.Context, client EKSClustersAPI, eksClusterName string) error {
	var allFargateProfiles []*string

	paginator := eks.NewListFargateProfilesPaginator(client, &eks.ListFargateProfilesInput{ClusterName: aws.String(eksClusterName)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		allFargateProfiles = append(allFargateProfiles, aws.StringSlice(page.FargateProfileNames)...)
	}

	// We can't delete Fargate profiles in parallel, so we delete them sequentially, waiting until the profile has been
	// deleted before moving on to the next one. This will make the delete routine very slow, but unfortunately, there
	// is no other alternative.
	// See https://docs.aws.amazon.com/eks/latest/APIReference/API_DeleteFargateProfile.html for more info on the serial
	// requirement.
	// Note that we aggregate deletion errors so that we at least attempt to delete all of them once.
	var allDeleteErrs error
	for _, fargateProfile := range allFargateProfiles {
		_, err := client.DeleteFargateProfile(ctx, &eks.DeleteFargateProfileInput{
			ClusterName:        aws.String(eksClusterName),
			FargateProfileName: fargateProfile,
		})
		if err != nil {
			logging.Debugf("[Failed] Failed deleting Fargate Profile %s associated with cluster %s: %s", aws.ToString(fargateProfile), eksClusterName, err)
			allDeleteErrs = multierror.Append(allDeleteErrs, err)
			continue
		}

		waiter := eks.NewFargateProfileDeletedWaiter(client)
		waitErr := waiter.Wait(ctx, &eks.DescribeFargateProfileInput{
			ClusterName:        aws.String(eksClusterName),
			FargateProfileName: fargateProfile,
		}, DefaultWaitTimeout)
		if waitErr != nil {
			logging.Debugf("[Failed] Failed waiting for Fargate Profile %s associated with cluster %s to be deleted: %s", aws.ToString(fargateProfile), eksClusterName, waitErr)
			allDeleteErrs = multierror.Append(allDeleteErrs, waitErr)
		} else {
			logging.Debugf("Deleted Fargate Profile %s associated with cluster %s", aws.ToString(fargateProfile), eksClusterName)
		}
	}

	return allDeleteErrs
}

// waitUntilEksClustersDeleted waits until the EKS cluster has been actually deleted from AWS. Returns a list of EKS
// cluster names that have been successfully deleted.
func waitUntilEksClustersDeleted(ctx context.Context, client EKSClustersAPI, resourceType string, eksClusterNames []*string) []*string {
	var successfullyDeleted []*string
	for _, eksClusterName := range eksClusterNames {
		waiter := eks.NewClusterDeletedWaiter(client)
		err := waiter.Wait(ctx, &eks.DescribeClusterInput{
			Name: eksClusterName,
		}, DefaultWaitTimeout)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(eksClusterName),
			ResourceType: resourceType,
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] Failed waiting for EKS cluster to be deleted %s: %s", *eksClusterName, err)
		} else {
			logging.Debugf("Deleted EKS cluster: %s", aws.ToString(eksClusterName))
			successfullyDeleted = append(successfullyDeleted, eksClusterName)
		}
	}
	return successfullyDeleted
}

// Custom errors

type TooManyEKSClustersErr struct{}

func (err TooManyEKSClustersErr) Error() string {
	return "Too many EKS Clusters requested at once."
}

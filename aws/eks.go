package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAllEksClusters returns a list of strings of EKS Cluster Names that uniquely identify each cluster.
func getAllEksClusters(awsSession *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := eks.New(awsSession)
	result, err := svc.ListClusters(&eks.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	filteredClusters, err := filterOutRecentEksClusters(svc, result.Clusters, excludeAfter)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return filteredClusters, nil
}

// filterOutRecentEksClusters will take in the list of clusters and filter out any clusters that were created after
// `excludeAfter`.
func filterOutRecentEksClusters(svc *eks.EKS, clusterNames []*string, excludeAfter time.Time) ([]*string, error) {
	var filteredEksClusterNames []*string
	for _, clusterName := range clusterNames {
		describeResult, err := svc.DescribeCluster(&eks.DescribeClusterInput{
			Name: clusterName,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		cluster := describeResult.Cluster
		if excludeAfter.After(*cluster.CreatedAt) {
			filteredEksClusterNames = append(filteredEksClusterNames, cluster.Name)
		}
	}
	return filteredEksClusterNames, nil
}

// deleteEksClusters deletes all clusters requested. Returns a list of cluster names that have been accepted by AWS
// for deletion.
func deleteEksClusters(svc *eks.EKS, eksClusterNames []*string) []*string {
	var requestedDeletes []*string
	for _, eksClusterName := range eksClusterNames {
		_, err := svc.DeleteCluster(&eks.DeleteClusterInput{Name: eksClusterName})
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed deleting EKS cluster %s: %s", *eksClusterName, err)
		} else {
			requestedDeletes = append(requestedDeletes, eksClusterName)
		}
	}
	return requestedDeletes
}

// waitUntilEksClustersDeleted waits until the EKS cluster has been actually deleted from AWS. Returns a list of EKS
// cluster names that have been successfully deleted.
func waitUntilEksClustersDeleted(svc *eks.EKS, eksClusterNames []*string) []*string {
	var successfullyDeleted []*string
	for _, eksClusterName := range eksClusterNames {
		err := svc.WaitUntilClusterDeleted(&eks.DescribeClusterInput{Name: eksClusterName})
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed waiting for EKS cluster to be deleted %s: %s", *eksClusterName, err)
		} else {
			logging.Logger.Infof("Deleted EKS cluster: %s", *eksClusterName)
			successfullyDeleted = append(successfullyDeleted, eksClusterName)
		}
	}
	return successfullyDeleted
}

// nukeAllEksClusters deletes all provided EKS clusters, waiting for them to be deleted before returning.
func nukeAllEksClusters(awsSession *session.Session, eksClusterNames []*string) error {
	numNuking := len(eksClusterNames)
	svc := eks.New(awsSession)

	if numNuking == 0 {
		logging.Logger.Infof("No EKS clusters to nuke in region %s", *awsSession.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting %d EKS clusters in region %s", numNuking, *awsSession.Config.Region)

	requestedDeletes := deleteEksClusters(svc, eksClusterNames)
	successfullyDeleted := waitUntilEksClustersDeleted(svc, requestedDeletes)

	numNuked := len(successfullyDeleted)
	logging.Logger.Infof("[OK] %d of %d EKS cluster(s) deleted in %s", numNuked, numNuking, *awsSession.Config.Region)
	return nil

}

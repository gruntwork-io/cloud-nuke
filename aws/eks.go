package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/collections"
	"github.com/gruntwork-io/go-commons/errors"
)

// The regions that support EKS. Refer to
// https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/
var eksRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-2",
	"eu-central-1",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"eu-north-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-south-1",
}

// eksSupportedRegion returns true if the provided region supports EKS
func eksSupportedRegion(region string) bool {
	return collections.ListContainsElement(eksRegions, region)
}

// getAllEksClusters returns a list of strings of EKS Cluster Names that uniquely identify each cluster.
func getAllEksClusters(awsSession *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := eks.New(awsSession)
	result, err := svc.ListClusters(&eks.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	excludedWithTagClusters, err := filterOutTaggedClusters(svc, result.Clusters)
	filteredClusters, err := filterOutRecentEksClusters(svc, excludedWithTagClusters, excludeAfter)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return filteredClusters, nil
}

// filterOutTaggedClusters removes clusters that have the tag set from the list of clusters to delete.
func filterOutTaggedClusters(svc *eks.EKS, clusterNames []*string) ([]*string, error) {
	var filteredEksClusterNames []*string
	for _, clusterName := range clusterNames {
		describeResult, err := svc.DescribeCluster(&eks.DescribeClusterInput{
			Name: clusterName,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		cluster := describeResult.Cluster
		if !hasClusterExcludeTag(cluster) {
			filteredEksClusterNames = append(filteredEksClusterNames, cluster.Name)
		}
	}
	return filteredEksClusterNames, nil
}

// hasClusterExcludeTag checks whether the exlude tag is set for a resource to skip deleting it.
func hasClusterExcludeTag(cluster *eks.Cluster) bool {
	// Exclude deletion of any buckets with cloud-nuke-excluded tags
	for k, v := range cluster.Tags {
		if k == AwsResourceExclusionTagKey && *v == "true" {
			return true
		}
	}
	return false
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

type ClusterNodegroup struct {
	ClusterName   *string
	NodegroupName *string
}

// deleteEksClusterNodeGroups deletes all node gorups in clusters requested. Returns a list of node group names that
// have been accepted by AWS for deletion.
func deleteEksClusterNodeGroups(svc *eks.EKS, eksClusterNames []*string) []*ClusterNodegroup {
	var requestedDeletes []*ClusterNodegroup
	for _, eksClusterName := range eksClusterNames {
		result, err := svc.ListNodegroups(&eks.ListNodegroupsInput{ClusterName: eksClusterName})
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed listing node groups for EKS cluster %s: %s", *eksClusterName, err)
			continue
		}

		for _, nodegroupName := range result.Nodegroups {
			_, err = svc.DeleteNodegroup(&eks.DeleteNodegroupInput{ClusterName: eksClusterName, NodegroupName: nodegroupName})
			if err != nil {
				logging.Logger.Errorf("[Failed] Failed deleting EKS node group %s in cluster %s: %s", *nodegroupName, *eksClusterName, err)
			} else {
				requestedDeletes = append(requestedDeletes, &ClusterNodegroup{eksClusterName, nodegroupName})
			}
		}
	}
	return requestedDeletes
}

// waitUntilEksClusterNodeGroupsDeleted waits until the EKS cluster node groups have been actually deleted from AWS.
// Returns a list of EKS cluster node groups that have been successfully deleted.
func waitUntilEksClusterNodeGroupsDeleted(svc *eks.EKS, clusterNodegroups []*ClusterNodegroup) []*ClusterNodegroup {
	var successfullyDeleted []*ClusterNodegroup
	for _, cn := range clusterNodegroups {
		err := svc.WaitUntilNodegroupDeleted(&eks.DescribeNodegroupInput{ClusterName: cn.ClusterName, NodegroupName: cn.NodegroupName})
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed waiting EKS node group %s in cluster %s: %s", *cn.NodegroupName, *cn.ClusterName, err)
		} else {
			successfullyDeleted = append(successfullyDeleted, cn)
		}
	}
	return successfullyDeleted
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

	deleteNodeGroupRequests := deleteEksClusterNodeGroups(svc, eksClusterNames)
	deletedNodeGroups := waitUntilEksClusterNodeGroupsDeleted(svc, deleteNodeGroupRequests)
	logging.Logger.Infof("[OK] %d of %d EKS cluster node group(s) deleted in %s",
		len(deletedNodeGroups), len(deleteNodeGroupRequests), *awsSession.Config.Region)

	requestedDeletes := deleteEksClusters(svc, eksClusterNames)
	successfullyDeleted := waitUntilEksClustersDeleted(svc, requestedDeletes)

	numNuked := len(successfullyDeleted)
	logging.Logger.Infof("[OK] %d of %d EKS cluster(s) deleted in %s", numNuked, numNuking, *awsSession.Config.Region)
	return nil

}

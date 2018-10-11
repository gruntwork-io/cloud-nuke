package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// getAllEcsClusters - Returns a string of ECS Cluster ARNs, which uniquely identifies the cluster.
// NOTE:
//   AWS api doesn't provide the necessary information to implement
//   `excludeAfter` filter at the cluster level, so we will implement it at the
//   service level. Clusters can't be deleted if they have active services
//   running on it. In practice this means that clusters that have recently
//   been used by launching services to it will not be deleted because it will
//   still have services running on it, since the services will be filtered out
//   using `excludeAfter`. However, recently created clusters could still be
//   deleted if it has not been used yet by deploying a service to it.
func getAllEcsClusters(awsSession *session.Session) ([]*string, error) {
	svc := ecs.New(awsSession)
	result, err := svc.ListClusters(&ecs.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return result.ClusterArns, nil
}

// Deletes all given ECS clusters. Note that this will swallow failed deletes
// and continue along, logging the cluster ARN so that we can find it later.
func nukeAllEcsClusters(awsSession *session.Session, ecsClusterArns []*string) error {
	numNuking := len(ecsClusterArns)
	svc := ecs.New(awsSession)

	if numNuking == 0 {
		logging.Logger.Infof("No ECS clusters to nuke in region %s", *awsSession.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting %d ECS clusters in region %s", numNuking, *awsSession.Config.Region)

	numNuked := 0
	for _, ecsClusterArn := range ecsClusterArns {
		params := &ecs.DeleteClusterInput{Cluster: ecsClusterArn}
		_, err := svc.DeleteCluster(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] Could not delete cluster %s: %s", *ecsClusterArn, err.Error())
		} else {
			logging.Logger.Infof("Deleted cluster: %s", *ecsClusterArn)
			numNuked += 1
		}
	}
	logging.Logger.Infof("[OK] %d of %d ECS cluster(s) deleted in %s", numNuked, numNuking, *awsSession.Config.Region)
	return nil
}

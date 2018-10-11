package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// filterOutRecentServices - Given a list of services and an excludeAfter
// timestamp, filter out any services that were created after `excludeAfter`.
func filterOutRecentServices(svc *ecs.ECS, clusterArn *string, ecsServiceArns []string, excludeAfter time.Time) ([]*string, error) {
	// Fetch descriptions in batches of 10, which is the max that AWS
	// accepts for describe service.
	var filteredEcsServiceArns []*string
	batches := split(ecsServiceArns, 10)
	for _, batch := range batches {
		params := &ecs.DescribeServicesInput{
			Cluster:  clusterArn,
			Services: awsgo.StringSlice(batch),
		}
		describeResult, err := svc.DescribeServices(params)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, service := range describeResult.Services {
			if excludeAfter.After(*service.CreatedAt) {
				filteredEcsServiceArns = append(filteredEcsServiceArns, service.ServiceArn)
			}
		}
	}
	return filteredEcsServiceArns, nil
}

// getAllEcsServices - Returns a formatted string of ECS Service ARNs, which
// uniquely identifies the service, in addition to a mapping of services to
// clusters. For ECS, need to track ECS clusters of services as all service
// level API endpoints require providing the corresponding cluster.
// Note that this looks up services by ECS cluster ARNs.
func getAllEcsServices(awsSession *session.Session, ecsClusterArns []*string, excludeAfter time.Time) ([]*string, map[string]string, error) {
	ecsServiceClusterMap := map[string]string{}
	svc := ecs.New(awsSession)

	// For each cluster, fetch all services, filtering out recently created
	// ones.
	var ecsServiceArns []*string
	for _, clusterArn := range ecsClusterArns {
		result, err := svc.ListServices(&ecs.ListServicesInput{Cluster: clusterArn})
		if err != nil {
			return nil, nil, errors.WithStackTrace(err)
		}
		filteredServiceArns, err := filterOutRecentServices(svc, clusterArn, awsgo.StringValueSlice(result.ServiceArns), excludeAfter)
		if err != nil {
			return nil, nil, errors.WithStackTrace(err)
		}
		// Update mapping to be used later in nuking
		for _, serviceArn := range filteredServiceArns {
			ecsServiceClusterMap[*serviceArn] = *clusterArn
		}
		ecsServiceArns = append(ecsServiceArns, filteredServiceArns...)
	}

	return ecsServiceArns, ecsServiceClusterMap, nil
}

// Deletes all provided ECS Services. At a high level this involves two steps:
// 1.) Drain all tasks from the service so that nothing is
//     running.
// 2.) Delete service object once no tasks are running.
// Note that this will swallow failed deletes and continue along, logging the
// service ARN so that we can find it later.
func nukeAllEcsServices(awsSession *session.Session, ecsServiceClusterMap map[string]string, ecsServiceArns []*string) error {
	numNuking := len(ecsServiceArns)
	svc := ecs.New(awsSession)

	if numNuking == 0 {
		logging.Logger.Infof("No ECS services to nuke in region %s", *awsSession.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting %d ECS services in region %s", numNuking, *awsSession.Config.Region)

	// First, drain all the services to 0. You can't delete a
	// service that is running tasks.
	// Note that we request all the drains at once, and then
	// wait for them in a separate loop because it will take a
	// while to drain the services.
	var requestedDrains []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.UpdateServiceInput{
			Cluster:      awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
			Service:      ecsServiceArn,
			DesiredCount: awsgo.Int64(0),
		}
		_, err := svc.UpdateService(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed to drain service %s: %s", *ecsServiceArn, err)
		} else {
			requestedDrains = append(requestedDrains, ecsServiceArn)
		}
	}

	// Wait until service is fully drained by waiting for
	// stability, which is defined as desiredCount ==
	// runningCount
	var successfullyDrained []*string
	for _, ecsServiceArn := range requestedDrains {
		params := &ecs.DescribeServicesInput{
			Cluster:  awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
			Services: []*string{ecsServiceArn},
		}
		err := svc.WaitUntilServicesStable(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed waiting for service to be stable %s: %s", *ecsServiceArn, err)
		} else {
			logging.Logger.Infof("Drained service: %s", *ecsServiceArn)
			successfullyDrained = append(successfullyDrained, ecsServiceArn)
		}
	}

	// Now delete the services that were successfully drained
	var requestedDeletes []*string
	for _, ecsServiceArn := range successfullyDrained {
		params := &ecs.DeleteServiceInput{
			Cluster: awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
			Service: ecsServiceArn,
		}
		_, err := svc.DeleteService(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed deleting service %s: %s", *ecsServiceArn, err)
		} else {
			requestedDeletes = append(requestedDeletes, ecsServiceArn)
		}
	}

	// Wait until services are deleted
	numNuked := 0
	for _, ecsServiceArn := range requestedDeletes {
		params := &ecs.DescribeServicesInput{
			Cluster:  awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
			Services: []*string{ecsServiceArn},
		}
		err := svc.WaitUntilServicesInactive(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed waiting for service to be deleted %s: %s", *ecsServiceArn, err)
		} else {
			logging.Logger.Infof("Deleted service: %s", *ecsServiceArn)
			numNuked += 1
		}
	}

	logging.Logger.Infof("[OK] %d of %d ECS service(s) deleted in %s", numNuked, numNuking, *awsSession.Config.Region)
	return nil
}

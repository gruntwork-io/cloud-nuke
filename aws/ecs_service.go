package aws

import (
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAllEcsClusters - Returns a string of ECS Cluster ARNs, which uniquely identifies the cluster.
// We need to get all clusters before we can get all services.
func getAllEcsClusters(awsSession *session.Session) ([]*string, error) {
	svc := ecs.New(awsSession)
	clusterArns := []*string{}
	result, err := svc.ListClusters(&ecs.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	clusterArns = append(clusterArns, result.ClusterArns...)

	// Handle pagination: continuously pull the next page if nextToken is set
	for awsgo.StringValue(result.NextToken) != "" {
		result, err = svc.ListClusters(&ecs.ListClustersInput{NextToken: result.NextToken})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		clusterArns = append(clusterArns, result.ClusterArns...)
	}

	return clusterArns, nil
}

// filterOutRecentServices - Given a list of services and an excludeAfter
// timestamp, filter out any services that were created after `excludeAfter.
// Additionally, filter based on Config file patterns.
func filterOutRecentServices(svc *ecs.ECS, clusterArn *string, ecsServiceArns []string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
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
			if shouldIncludeECSService(service, excludeAfter, configObj) {
				filteredEcsServiceArns = append(filteredEcsServiceArns, service.ServiceArn)
			}
		}
	}
	return filteredEcsServiceArns, nil
}

func shouldIncludeECSService(service *ecs.Service, excludeAfter time.Time, configObj config.Config) bool {
	if service == nil {
		return false
	}

	if service.CreatedAt != nil && excludeAfter.Before(*service.CreatedAt) {
		return false
	}

	return config.ShouldInclude(
		awsgo.StringValue(service.ServiceName),
		configObj.ECSService.IncludeRule.NamesRegExp,
		configObj.ECSService.ExcludeRule.NamesRegExp,
	)
}

// getAllEcsServices - Returns a formatted string of ECS Service ARNs, which
// uniquely identifies the service, in addition to a mapping of services to
// clusters. For ECS, need to track ECS clusters of services as all service
// level API endpoints require providing the corresponding cluster.
// Note that this looks up services by ECS cluster ARNs.
func getAllEcsServices(awsSession *session.Session, ecsClusterArns []*string, excludeAfter time.Time, configObj config.Config) ([]*string, map[string]string, error) {
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
		filteredServiceArns, err := filterOutRecentServices(svc, clusterArn, awsgo.StringValueSlice(result.ServiceArns), excludeAfter, configObj)
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

// drainEcsServices - Drain all tasks from all services requested. This will
// return a list of service ARNs that have been successfully requested to be
// drained.
func drainEcsServices(svc *ecs.ECS, ecsServiceClusterMap map[string]string, ecsServiceArns []*string) []*string {
	var requestedDrains []*string
	for _, ecsServiceArn := range ecsServiceArns {

		describeParams := &ecs.DescribeServicesInput{
			Cluster:  awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
			Services: []*string{ecsServiceArn},
		}
		describeServicesOutput, err := svc.DescribeServices(describeParams)
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed to describe service %s: %s", *ecsServiceArn, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Service",
			}, map[string]interface{}{
				"region": *svc.Config.Region,
				"reason": "Unable to describe",
			})
		} else {
			schedulingStrategy := *describeServicesOutput.Services[0].SchedulingStrategy
			if schedulingStrategy != "DAEMON" {
				params := &ecs.UpdateServiceInput{
					Cluster:      awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
					Service:      ecsServiceArn,
					DesiredCount: awsgo.Int64(0),
				}
				_, err := svc.UpdateService(params)
				if err != nil {
					logging.Logger.Errorf("[Failed] Failed to drain service %s: %s", *ecsServiceArn, err)
					telemetry.TrackEvent(commonTelemetry.EventContext{
						EventName: "Error Nuking ECS Service",
					}, map[string]interface{}{
						"region": *svc.Config.Region,
						"reason": "Unable to drain",
					})
				}
			} else {
				requestedDrains = append(requestedDrains, ecsServiceArn)
			}
		}
	}
	return requestedDrains
}

// waitUntilServiceDrained - Waits until all tasks have been drained from the
// given list of services, by waiting for stability which is defined as
// desiredCount == runningCount. This will return a list of service ARNs that
// have successfully been drained.
func waitUntilServicesDrained(svc *ecs.ECS, ecsServiceClusterMap map[string]string, ecsServiceArns []*string) []*string {
	var successfullyDrained []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DescribeServicesInput{
			Cluster:  awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
			Services: []*string{ecsServiceArn},
		}
		err := svc.WaitUntilServicesStable(params)
		if err != nil {
			logging.Logger.Debugf("[Failed] Failed waiting for service to be stable %s: %s", *ecsServiceArn, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Service",
			}, map[string]interface{}{
				"region": *svc.Config.Region,
				"reason": "Failed Waiting for Drain",
			})
		} else {
			logging.Logger.Debugf("Drained service: %s", *ecsServiceArn)
			successfullyDrained = append(successfullyDrained, ecsServiceArn)
		}
	}
	return successfullyDrained
}

// deleteEcsServices - Deletes all services requested. Returns a list of
// service ARNs that have been accepted by AWS for deletion.
func deleteEcsServices(svc *ecs.ECS, ecsServiceClusterMap map[string]string, ecsServiceArns []*string) []*string {
	var requestedDeletes []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DeleteServiceInput{
			Cluster: awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
			Service: ecsServiceArn,
		}
		_, err := svc.DeleteService(params)
		if err != nil {
			logging.Logger.Debugf("[Failed] Failed deleting service %s: %s", *ecsServiceArn, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Service",
			}, map[string]interface{}{
				"region": *svc.Config.Region,
				"reason": "Unable to Delete",
			})
		} else {
			requestedDeletes = append(requestedDeletes, ecsServiceArn)
		}
	}
	return requestedDeletes
}

// waitUntilServicesDeleted - Waits until the service has been actually deleted
// from AWS. Returns a list of service ARNs that have been successfully
// deleted.
func waitUntilServicesDeleted(svc *ecs.ECS, ecsServiceClusterMap map[string]string, ecsServiceArns []*string) []*string {
	var successfullyDeleted []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DescribeServicesInput{
			Cluster:  awsgo.String(ecsServiceClusterMap[*ecsServiceArn]),
			Services: []*string{ecsServiceArn},
		}
		err := svc.WaitUntilServicesInactive(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(ecsServiceArn),
			ResourceType: "ECS Service",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] Failed waiting for service to be deleted %s: %s", *ecsServiceArn, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Service",
			}, map[string]interface{}{
				"region": *svc.Config.Region,
				"reason": "Failed Waiting for Delete",
			})
		} else {
			logging.Logger.Debugf("Deleted service: %s", *ecsServiceArn)
			successfullyDeleted = append(successfullyDeleted, ecsServiceArn)
		}
	}
	return successfullyDeleted
}

// Deletes all provided ECS Services. At a high level this involves two steps:
// 1.) Drain all tasks from the service so that nothing is
//
//	running.
//
// 2.) Delete service object once no tasks are running.
// Note that this will swallow failed deletes and continue along, logging the
// service ARN so that we can find it later.
func nukeAllEcsServices(awsSession *session.Session, ecsServiceClusterMap map[string]string, ecsServiceArns []*string) error {
	numNuking := len(ecsServiceArns)
	svc := ecs.New(awsSession)

	if numNuking == 0 {
		logging.Logger.Debugf("No ECS services to nuke in region %s", *awsSession.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting %d ECS services in region %s", numNuking, *awsSession.Config.Region)

	// First, drain all the services to 0. You can't delete a
	// service that is running tasks.
	// Note that we request all the drains at once, and then
	// wait for them in a separate loop because it will take a
	// while to drain the services.
	// Then, we delete the services that have been successfully drained.
	requestedDrains := drainEcsServices(svc, ecsServiceClusterMap, ecsServiceArns)
	successfullyDrained := waitUntilServicesDrained(svc, ecsServiceClusterMap, requestedDrains)
	requestedDeletes := deleteEcsServices(svc, ecsServiceClusterMap, successfullyDrained)
	successfullyDeleted := waitUntilServicesDeleted(svc, ecsServiceClusterMap, requestedDeletes)

	numNuked := len(successfullyDeleted)
	logging.Logger.Debugf("[OK] %d of %d ECS service(s) deleted in %s", numNuked, numNuking, *awsSession.Config.Region)
	return nil
}

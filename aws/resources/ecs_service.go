package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/andrewderr/cloud-nuke-a1/util"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAllEcsClusters - Returns a string of ECS Cluster ARNs, which uniquely identifies the cluster.
// We need to get all clusters before we can get all services.
func (services *ECSServices) getAllEcsClusters() ([]*string, error) {
	clusterArns := []*string{}
	result, err := services.Client.ListClusters(&ecs.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	clusterArns = append(clusterArns, result.ClusterArns...)

	// Handle pagination: continuously pull the next page if nextToken is set
	for awsgo.StringValue(result.NextToken) != "" {
		result, err = services.Client.ListClusters(&ecs.ListClustersInput{NextToken: result.NextToken})
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
func (services *ECSServices) filterOutRecentServices(clusterArn *string, ecsServiceArns []string, configObj config.Config) ([]*string, error) {
	// Fetch descriptions in batches of 10, which is the max that AWS
	// accepts for describe service.
	var filteredEcsServiceArns []*string
	batches := util.Split(ecsServiceArns, 10)
	for _, batch := range batches {
		params := &ecs.DescribeServicesInput{
			Cluster:  clusterArn,
			Services: awsgo.StringSlice(batch),
		}
		describeResult, err := services.Client.DescribeServices(params)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, service := range describeResult.Services {
			if configObj.ECSService.ShouldInclude(config.ResourceValue{
				Name: service.ServiceName,
				Time: service.CreatedAt,
			}) {
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
func (services *ECSServices) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	ecsClusterArns, err := services.getAllEcsClusters()
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	ecsServiceClusterMap := map[string]string{}

	// For each cluster, fetch all services, filtering out recently created
	// ones.
	var ecsServiceArns []*string
	for _, clusterArn := range ecsClusterArns {
		result, err := services.Client.ListServices(&ecs.ListServicesInput{Cluster: clusterArn})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		filteredServiceArns, err := services.filterOutRecentServices(clusterArn, awsgo.StringValueSlice(result.ServiceArns), configObj)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		// Update mapping to be used later in nuking
		for _, serviceArn := range filteredServiceArns {
			ecsServiceClusterMap[*serviceArn] = *clusterArn
		}
		ecsServiceArns = append(ecsServiceArns, filteredServiceArns...)
	}

	services.ServiceClusterMap = ecsServiceClusterMap
	return ecsServiceArns, nil
}

// drainEcsServices - Drain all tasks from all services requested. This will
// return a list of service ARNs that have been successfully requested to be
// drained.
func (services *ECSServices) drainEcsServices(ecsServiceArns []*string) []*string {
	var requestedDrains []*string
	for _, ecsServiceArn := range ecsServiceArns {

		describeParams := &ecs.DescribeServicesInput{
			Cluster:  awsgo.String(services.ServiceClusterMap[*ecsServiceArn]),
			Services: []*string{ecsServiceArn},
		}
		describeServicesOutput, err := services.Client.DescribeServices(describeParams)
		if err != nil {
			logging.Errorf("[Failed] Failed to describe service %s: %s", *ecsServiceArn, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Service",
			}, map[string]interface{}{
				"region": services.Region,
				"reason": "Unable to describe",
			})
		} else {

			schedulingStrategy := *describeServicesOutput.Services[0].SchedulingStrategy
			if schedulingStrategy == "DAEMON" {
				requestedDrains = append(requestedDrains, ecsServiceArn)
			} else {
				params := &ecs.UpdateServiceInput{
					Cluster:      awsgo.String(services.ServiceClusterMap[*ecsServiceArn]),
					Service:      ecsServiceArn,
					DesiredCount: awsgo.Int64(0),
				}
				_, err = services.Client.UpdateService(params)
				if err != nil {
					logging.Errorf("[Failed] Failed to drain service %s: %s", *ecsServiceArn, err)
					telemetry.TrackEvent(commonTelemetry.EventContext{
						EventName: "Error Nuking ECS Service",
					}, map[string]interface{}{
						"region": services.Region,
						"reason": "Unable to drain",
					})
				} else {
					requestedDrains = append(requestedDrains, ecsServiceArn)
				}
			}
		}
	}
	return requestedDrains
}

// waitUntilServiceDrained - Waits until all tasks have been drained from the
// given list of services, by waiting for stability which is defined as
// desiredCount == runningCount. This will return a list of service ARNs that
// have successfully been drained.
func (services *ECSServices) waitUntilServicesDrained(ecsServiceArns []*string) []*string {
	var successfullyDrained []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DescribeServicesInput{
			Cluster:  awsgo.String(services.ServiceClusterMap[*ecsServiceArn]),
			Services: []*string{ecsServiceArn},
		}
		err := services.Client.WaitUntilServicesStable(params)
		if err != nil {
			logging.Debugf("[Failed] Failed waiting for service to be stable %s: %s", *ecsServiceArn, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Service",
			}, map[string]interface{}{
				"region": services.Region,
				"reason": "Failed Waiting for Drain",
			})
		} else {
			logging.Debugf("Drained service: %s", *ecsServiceArn)
			successfullyDrained = append(successfullyDrained, ecsServiceArn)
		}
	}
	return successfullyDrained
}

// deleteEcsServices - Deletes all services requested. Returns a list of
// service ARNs that have been accepted by AWS for deletion.
func (services *ECSServices) deleteEcsServices(ecsServiceArns []*string) []*string {
	var requestedDeletes []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DeleteServiceInput{
			Cluster: awsgo.String(services.ServiceClusterMap[*ecsServiceArn]),
			Service: ecsServiceArn,
		}
		_, err := services.Client.DeleteService(params)
		if err != nil {
			logging.Debugf("[Failed] Failed deleting service %s: %s", *ecsServiceArn, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Service",
			}, map[string]interface{}{
				"region": services.Region,
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
func (services *ECSServices) waitUntilServicesDeleted(ecsServiceArns []*string) []*string {
	var successfullyDeleted []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DescribeServicesInput{
			Cluster:  awsgo.String(services.ServiceClusterMap[*ecsServiceArn]),
			Services: []*string{ecsServiceArn},
		}
		err := services.Client.WaitUntilServicesInactive(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(ecsServiceArn),
			ResourceType: "ECS Service",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] Failed waiting for service to be deleted %s: %s", *ecsServiceArn, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Service",
			}, map[string]interface{}{
				"region": services.Region,
				"reason": "Failed Waiting for Delete",
			})
		} else {
			logging.Debugf("Deleted service: %s", *ecsServiceArn)
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
func (services *ECSServices) nukeAll(ecsServiceArns []*string) error {
	numNuking := len(ecsServiceArns)
	if numNuking == 0 {
		logging.Debugf("No ECS services to nuke in region %s", services.Region)
		return nil
	}

	logging.Debugf("Deleting %d ECS services in region %s", numNuking, services.Region)

	// First, drain all the services to 0. You can't delete a
	// service that is running tasks.
	// Note that we request all the drains at once, and then
	// wait for them in a separate loop because it will take a
	// while to drain the services.
	// Then, we delete the services that have been successfully drained.
	requestedDrains := services.drainEcsServices(ecsServiceArns)
	successfullyDrained := services.waitUntilServicesDrained(requestedDrains)
	requestedDeletes := services.deleteEcsServices(successfullyDrained)
	successfullyDeleted := services.waitUntilServicesDeleted(requestedDeletes)

	numNuked := len(successfullyDeleted)
	logging.Debugf("[OK] %d of %d ECS service(s) deleted in %s", numNuked, numNuking, services.Client)
	return nil
}

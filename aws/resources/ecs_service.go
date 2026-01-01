package resources

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// ECSServicesAPI defines the interface for ECS Services operations.
type ECSServicesAPI interface {
	ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListServices(ctx context.Context, params *ecs.ListServicesInput, optFns ...func(*ecs.Options)) (*ecs.ListServicesOutput, error)
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	DeleteService(ctx context.Context, params *ecs.DeleteServiceInput, optFns ...func(*ecs.Options)) (*ecs.DeleteServiceOutput, error)
	UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
}

// ecsServicesState holds state that needs to be shared between list and nuke phases.
type ecsServicesState struct {
	mu                sync.Mutex
	serviceClusterMap map[string]string
	timeout           time.Duration
}

// globalECSServicesState is the global state for ECS Services operations.
var globalECSServicesState = &ecsServicesState{
	serviceClusterMap: make(map[string]string),
	timeout:           DefaultWaitTimeout,
}

// NewECSServices creates a new ECSServices resource using the generic resource pattern.
func NewECSServices() AwsResource {
	return NewAwsResource(&resource.Resource[ECSServicesAPI]{
		ResourceTypeName: "ecsserv",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[ECSServicesAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for ECS client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ecs.NewFromConfig(awsCfg)
			// Reset global state on init
			globalECSServicesState.mu.Lock()
			globalECSServicesState.serviceClusterMap = make(map[string]string)
			globalECSServicesState.timeout = DefaultWaitTimeout
			globalECSServicesState.mu.Unlock()
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ECSService
		},
		Lister: listECSServices,
		Nuker:  deleteECSServices,
	})
}

// getAllEcsClusterArnsForServices - Returns a string of ECS Cluster ARNs, which uniquely identifies the cluster.
// We need to get all clusters before we can get all services.
func getAllEcsClusterArnsForServices(ctx context.Context, client ECSServicesAPI) ([]*string, error) {
	var clusterArns []string
	result, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	clusterArns = append(clusterArns, result.ClusterArns...)

	// Handle pagination: continuously pull the next page if nextToken is set
	for aws.ToString(result.NextToken) != "" {
		result, err = client.ListClusters(ctx, &ecs.ListClustersInput{NextToken: result.NextToken})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		clusterArns = append(clusterArns, result.ClusterArns...)
	}

	return aws.StringSlice(clusterArns), nil
}

// filterOutRecentServices - Given a list of services and an excludeAfter
// timestamp, filter out any services that were created after `excludeAfter.
// Additionally, filter based on Config file patterns.
func filterOutRecentServices(ctx context.Context, client ECSServicesAPI, clusterArn *string, ecsServiceArns []string, cfg config.ResourceType) ([]*string, error) {
	// Fetch descriptions in batches of 10, which is the max that AWS
	// accepts for describe service.
	var filteredEcsServiceArns []*string
	batches := util.Split(ecsServiceArns, 10)
	for _, batch := range batches {
		params := &ecs.DescribeServicesInput{
			Cluster:  clusterArn,
			Services: batch,
			Include:  []types.ServiceField{types.ServiceFieldTags},
		}
		describeResult, err := client.DescribeServices(ctx, params)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, service := range describeResult.Services {
			tags := make(map[string]string)
			for _, tag := range service.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: service.ServiceName,
				Time: service.CreatedAt,
				Tags: tags,
			}) {
				filteredEcsServiceArns = append(filteredEcsServiceArns, service.ServiceArn)
			}
		}
	}
	return filteredEcsServiceArns, nil
}

// listECSServices - Returns a formatted string of ECS Service ARNs, which
// uniquely identifies the service, in addition to a mapping of services to
// clusters. For ECS, need to track ECS clusters of services as all service
// level API endpoints require providing the corresponding cluster.
// Note that this looks up services by ECS cluster ARNs.
func listECSServices(ctx context.Context, client ECSServicesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	ecsClusterArns, err := getAllEcsClusterArnsForServices(ctx, client)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	ecsServiceClusterMap := map[string]string{}

	// For each cluster, fetch all services, filtering out recently created
	// ones.
	var ecsServiceArns []*string
	for _, clusterArn := range ecsClusterArns {
		result, err := client.ListServices(ctx, &ecs.ListServicesInput{Cluster: clusterArn})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		filteredServiceArns, err := filterOutRecentServices(ctx, client, clusterArn, result.ServiceArns, cfg)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		// Update mapping to be used later in nuking
		for _, serviceArn := range filteredServiceArns {
			ecsServiceClusterMap[*serviceArn] = *clusterArn
		}
		ecsServiceArns = append(ecsServiceArns, filteredServiceArns...)
	}

	// Store mapping in global state for use during nuke phase
	globalECSServicesState.mu.Lock()
	globalECSServicesState.serviceClusterMap = ecsServiceClusterMap
	globalECSServicesState.mu.Unlock()

	return ecsServiceArns, nil
}

// drainEcsServices - Drain all tasks from all services requested. This will
// return a list of service ARNs that have been successfully requested to be
// drained.
func drainEcsServices(ctx context.Context, client ECSServicesAPI, ecsServiceArns []*string, serviceClusterMap map[string]string) []*string {
	var requestedDrains []*string
	for _, ecsServiceArn := range ecsServiceArns {

		describeParams := &ecs.DescribeServicesInput{
			Cluster:  aws.String(serviceClusterMap[*ecsServiceArn]),
			Services: []string{*ecsServiceArn},
		}
		describeServicesOutput, err := client.DescribeServices(ctx, describeParams)
		if err != nil {
			logging.Errorf("[Failed] Failed to describe service %s: %s", *ecsServiceArn, err)
		} else {

			schedulingStrategy := describeServicesOutput.Services[0].SchedulingStrategy
			if schedulingStrategy == types.SchedulingStrategyDaemon {
				requestedDrains = append(requestedDrains, ecsServiceArn)
			} else {
				params := &ecs.UpdateServiceInput{
					Cluster:      aws.String(serviceClusterMap[*ecsServiceArn]),
					Service:      ecsServiceArn,
					DesiredCount: aws.Int32(0),
				}
				_, err = client.UpdateService(ctx, params)
				if err != nil {
					logging.Errorf("[Failed] Failed to drain service %s: %s", *ecsServiceArn, err)
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
func waitUntilServicesDrained(ctx context.Context, client ECSServicesAPI, ecsServiceArns []*string, serviceClusterMap map[string]string, timeout time.Duration) []*string {
	var successfullyDrained []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DescribeServicesInput{
			Cluster:  aws.String(serviceClusterMap[*ecsServiceArn]),
			Services: []string{*ecsServiceArn},
		}

		waiter := ecs.NewServicesStableWaiter(client)
		err := waiter.Wait(ctx, params, timeout)
		if err != nil {
			logging.Debugf("[Failed] Failed waiting for service to be stable %s: %s", *ecsServiceArn, err)
		} else {
			logging.Debugf("Drained service: %s", *ecsServiceArn)
			successfullyDrained = append(successfullyDrained, ecsServiceArn)
		}
	}
	return successfullyDrained
}

// deleteEcsServicesIndividually - Deletes all services requested. Returns a list of
// service ARNs that have been accepted by AWS for deletion.
func deleteEcsServicesIndividually(ctx context.Context, client ECSServicesAPI, ecsServiceArns []*string, serviceClusterMap map[string]string) []*string {
	var requestedDeletes []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DeleteServiceInput{
			Cluster: aws.String(serviceClusterMap[*ecsServiceArn]),
			Service: ecsServiceArn,
		}
		_, err := client.DeleteService(ctx, params)
		if err != nil {
			logging.Debugf("[Failed] Failed deleting service %s: %s", *ecsServiceArn, err)
		} else {
			requestedDeletes = append(requestedDeletes, ecsServiceArn)
		}
	}
	return requestedDeletes
}

// waitUntilServicesDeleted - Waits until the service has been actually deleted
// from AWS. Returns a list of service ARNs that have been successfully
// deleted.
func waitUntilServicesDeleted(ctx context.Context, client ECSServicesAPI, ecsServiceArns []*string, serviceClusterMap map[string]string, timeout time.Duration) []*string {
	var successfullyDeleted []*string
	for _, ecsServiceArn := range ecsServiceArns {
		params := &ecs.DescribeServicesInput{
			Cluster:  aws.String(serviceClusterMap[*ecsServiceArn]),
			Services: []string{*ecsServiceArn},
		}

		waiter := ecs.NewServicesInactiveWaiter(client)
		err := waiter.Wait(ctx, params, timeout)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(ecsServiceArn),
			ResourceType: "ECS Service",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] Failed waiting for service to be deleted %s: %s", *ecsServiceArn, err)
		} else {
			logging.Debugf("Deleted service: %s", *ecsServiceArn)
			successfullyDeleted = append(successfullyDeleted, ecsServiceArn)
		}
	}
	return successfullyDeleted
}

// deleteECSServices deletes all provided ECS Services. At a high level this involves two steps:
// 1.) Drain all tasks from the service so that nothing is running.
// 2.) Delete service object once no tasks are running.
// Note that this will swallow failed deletes and continue along, logging the
// service ARN so that we can find it later.
func deleteECSServices(ctx context.Context, client ECSServicesAPI, scope resource.Scope, resourceType string, ecsServiceArns []*string) error {
	numNuking := len(ecsServiceArns)
	if numNuking == 0 {
		logging.Debugf("No ECS services to nuke in region %s", scope.Region)
		return nil
	}

	// Get service cluster map from global state
	globalECSServicesState.mu.Lock()
	serviceClusterMap := globalECSServicesState.serviceClusterMap
	timeout := globalECSServicesState.timeout
	globalECSServicesState.mu.Unlock()

	logging.Debugf("Deleting %d ECS services in region %s", numNuking, scope.Region)

	// First, drain all the services to 0. You can't delete a
	// service that is running tasks.
	// Note that we request all the drains at once, and then
	// wait for them in a separate loop because it will take a
	// while to drain the services.
	// Then, we delete the services that have been successfully drained.
	requestedDrains := drainEcsServices(ctx, client, ecsServiceArns, serviceClusterMap)
	successfullyDrained := waitUntilServicesDrained(ctx, client, requestedDrains, serviceClusterMap, timeout)
	requestedDeletes := deleteEcsServicesIndividually(ctx, client, successfullyDrained, serviceClusterMap)
	successfullyDeleted := waitUntilServicesDeleted(ctx, client, requestedDeletes, serviceClusterMap, timeout)

	numNuked := len(successfullyDeleted)
	logging.Debugf("[OK] %d of %d ECS service(s) deleted in %s", numNuked, numNuking, scope.Region)
	return nil
}

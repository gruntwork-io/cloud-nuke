package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
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

// ecsServicesResource holds state needed for ECS service operations.
type ecsServicesResource struct {
	*resource.Resource[ECSServicesAPI]
	serviceClusterMap map[string]string
}

// NewECSServices creates a new ECSServices resource using the generic resource pattern.
func NewECSServices() AwsResource {
	r := &ecsServicesResource{
		Resource: &resource.Resource[ECSServicesAPI]{
			ResourceTypeName: "ecsserv",
			BatchSize:        49,
		},
		serviceClusterMap: make(map[string]string),
	}

	r.InitClient = WrapAwsInitClient(func(res *resource.Resource[ECSServicesAPI], cfg aws.Config) {
		res.Scope.Region = cfg.Region
		res.Client = ecs.NewFromConfig(cfg)
		// Reset state on init
		r.serviceClusterMap = make(map[string]string)
	})

	r.ConfigGetter = func(c config.Config) config.ResourceType {
		return c.ECSService
	}

	r.Lister = func(ctx context.Context, client ECSServicesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
		return listECSServices(ctx, client, scope, cfg, r.serviceClusterMap)
	}

	r.Nuker = func(ctx context.Context, client ECSServicesAPI, scope resource.Scope, resourceType string, ids []*string) []resource.NukeResult {
		return deleteECSServices(ctx, client, scope, resourceType, ids, r.serviceClusterMap)
	}

	return &AwsResourceAdapter[ECSServicesAPI]{Resource: r.Resource}
}

// listECSServices returns all ECS Service ARNs and populates the service-to-cluster mapping.
func listECSServices(ctx context.Context, client ECSServicesAPI, scope resource.Scope, cfg config.ResourceType, serviceClusterMap map[string]string) ([]*string, error) {
	// Get all cluster ARNs
	clusterArns, err := listAllECSClusterArns(ctx, client)
	if err != nil {
		return nil, err
	}

	var serviceArns []*string
	for _, clusterArn := range clusterArns {
		paginator := ecs.NewListServicesPaginator(client, &ecs.ListServicesInput{Cluster: clusterArn})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			filtered, err := filterECSServices(ctx, client, clusterArn, page.ServiceArns, cfg)
			if err != nil {
				return nil, err
			}

			for _, arn := range filtered {
				serviceClusterMap[aws.ToString(arn)] = aws.ToString(clusterArn)
			}
			serviceArns = append(serviceArns, filtered...)
		}
	}

	return serviceArns, nil
}

// listAllECSClusterArns returns all ECS cluster ARNs.
func listAllECSClusterArns(ctx context.Context, client ECSServicesAPI) ([]*string, error) {
	var clusterArns []*string
	paginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		clusterArns = append(clusterArns, aws.StringSlice(page.ClusterArns)...)
	}
	return clusterArns, nil
}

// filterECSServices filters services based on config rules.
// DescribeServices accepts max 10 services per call.
func filterECSServices(ctx context.Context, client ECSServicesAPI, clusterArn *string, serviceArns []string, cfg config.ResourceType) ([]*string, error) {
	var filtered []*string
	batches := util.Split(serviceArns, 10)

	for _, batch := range batches {
		output, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  clusterArn,
			Services: batch,
			Include:  []types.ServiceField{types.ServiceFieldTags},
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, svc := range output.Services {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: svc.ServiceName,
				Time: svc.CreatedAt,
				Tags: convertECSTagsToMap(svc.Tags),
			}) {
				filtered = append(filtered, svc.ServiceArn)
			}
		}
	}
	return filtered, nil
}

func convertECSTagsToMap(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

// deleteECSServices deletes all provided ECS Services.
// For REPLICA services: scales to 0 first, then deletes.
// For DAEMON services: uses Force=true to delete directly.
func deleteECSServices(ctx context.Context, client ECSServicesAPI, scope resource.Scope, resourceType string, serviceArns []*string, serviceClusterMap map[string]string) []resource.NukeResult {
	if len(serviceArns) == 0 {
		return nil
	}

	logging.Infof("Deleting %d %s in %s", len(serviceArns), resourceType, scope)

	results := make([]resource.NukeResult, 0, len(serviceArns))
	for _, arn := range serviceArns {
		err := deleteECSService(ctx, client, arn, serviceClusterMap)
		results = append(results, resource.NukeResult{
			Identifier: aws.ToString(arn),
			Error:      err,
		})
	}

	return results
}

// deleteECSService deletes a single ECS service.
func deleteECSService(ctx context.Context, client ECSServicesAPI, serviceArn *string, serviceClusterMap map[string]string) error {
	clusterArn := serviceClusterMap[aws.ToString(serviceArn)]
	if clusterArn == "" {
		return errors.WithStackTrace(errClusterNotFound{serviceArn: aws.ToString(serviceArn)})
	}

	// Get service details to check scheduling strategy
	describeOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterArn),
		Services: []string{aws.ToString(serviceArn)},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(describeOutput.Services) == 0 {
		// Service already deleted
		return nil
	}

	svc := describeOutput.Services[0]

	// For REPLICA services, scale down first for graceful shutdown
	if svc.SchedulingStrategy == types.SchedulingStrategyReplica {
		if _, err := client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterArn),
			Service:      serviceArn,
			DesiredCount: aws.Int32(0),
		}); err != nil {
			logging.Debugf("Failed to scale down service %s, will force delete: %v", aws.ToString(serviceArn), err)
		}
	}

	// Delete with Force=true handles both scaled and non-scaled services
	// Force allows deletion even if tasks are still running
	_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
		Cluster: aws.String(clusterArn),
		Service: serviceArn,
		Force:   aws.Bool(true),
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Wait for service to become inactive
	waiter := ecs.NewServicesInactiveWaiter(client)
	if err := waiter.Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterArn),
		Services: []string{aws.ToString(serviceArn)},
	}, DefaultWaitTimeout); err != nil {
		logging.Debugf("Timeout waiting for service %s to become inactive: %v", aws.ToString(serviceArn), err)
		// Don't fail - the delete was initiated successfully
	}

	return nil
}

type errClusterNotFound struct {
	serviceArn string
}

func (e errClusterNotFound) Error() string {
	return "cluster not found for service: " + e.serviceArn
}

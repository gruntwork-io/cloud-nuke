package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudMapServicesAPI defines the interface for AWS Cloud Map API operations.
type CloudMapServicesAPI interface {
	ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error)
	DeleteService(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error)
	ListInstances(ctx context.Context, params *servicediscovery.ListInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListInstancesOutput, error)
	DeregisterInstance(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error)
	ListTagsForResource(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error)
}

// NewCloudMapServices creates a new CloudMapServices resource using the generic resource pattern.
func NewCloudMapServices() AwsResource {
	return NewAwsResource(&resource.Resource[CloudMapServicesAPI]{
		ResourceTypeName: "cloudmap-service",
		BatchSize:        50,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[CloudMapServicesAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = servicediscovery.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudMapService
		},
		Lister: listCloudMapServices,
		Nuker:  resource.SequentialDeleter(deleteCloudMapService),
	})
}

// listCloudMapServices retrieves all Cloud Map services matching the config filters.
func listCloudMapServices(ctx context.Context, client CloudMapServicesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var serviceIds []*string

	paginator := servicediscovery.NewListServicesPaginator(client, &servicediscovery.ListServicesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, service := range page.Services {
			tags, err := getCloudMapServiceTags(ctx, client, service.Arn)
			if err != nil {
				logging.Debugf("Error getting tags for Cloud Map service %s: %s", aws.ToString(service.Id), err)
				// Continue without tags rather than failing entirely
				tags = nil
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: service.Name,
				Time: service.CreateDate,
				Tags: tags,
			}) {
				serviceIds = append(serviceIds, service.Id)
			}
		}
	}

	return serviceIds, nil
}

// deleteCloudMapService deletes a single Cloud Map service after deregistering all its instances.
func deleteCloudMapService(ctx context.Context, client CloudMapServicesAPI, serviceId *string) error {
	// Step 1: Deregister all instances
	if err := deregisterCloudMapInstances(ctx, client, serviceId); err != nil {
		return err
	}

	// Step 2: Wait for instances to be fully deregistered
	if err := waitForCloudMapInstanceDeregistration(ctx, client, serviceId); err != nil {
		return err
	}

	// Step 3: Delete the service
	_, err := client.DeleteService(ctx, &servicediscovery.DeleteServiceInput{
		Id: serviceId,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// deregisterCloudMapInstances removes all instances from a Cloud Map service.
func deregisterCloudMapInstances(ctx context.Context, client CloudMapServicesAPI, serviceId *string) error {
	paginator := servicediscovery.NewListInstancesPaginator(client, &servicediscovery.ListInstancesInput{
		ServiceId: serviceId,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, instance := range page.Instances {
			logging.Debugf("Deregistering instance %s from service %s", aws.ToString(instance.Id), aws.ToString(serviceId))
			_, err := client.DeregisterInstance(ctx, &servicediscovery.DeregisterInstanceInput{
				ServiceId:  serviceId,
				InstanceId: instance.Id,
			})
			if err != nil {
				// Log but continue - instance may already be deregistered
				logging.Debugf("Error deregistering instance %s: %s", aws.ToString(instance.Id), err)
			}
		}
	}

	return nil
}

// waitForCloudMapInstanceDeregistration waits for all instances to be deregistered from a service.
func waitForCloudMapInstanceDeregistration(ctx context.Context, client CloudMapServicesAPI, serviceId *string) error {
	const (
		maxRetries        = 30
		sleepBetweenPolls = 5 * time.Second
	)

	for i := 0; i < maxRetries; i++ {
		hasInstances, err := serviceHasInstances(ctx, client, serviceId)
		if err != nil {
			return err
		}

		if !hasInstances {
			return nil
		}

		logging.Debugf("Waiting for instances in service %s to be deregistered (attempt %d/%d)",
			aws.ToString(serviceId), i+1, maxRetries)
		time.Sleep(sleepBetweenPolls)
	}

	return errors.WithStackTrace(fmt.Errorf("timeout waiting for instances to be deregistered in service %s", aws.ToString(serviceId)))
}

// serviceHasInstances checks if a Cloud Map service has any registered instances.
func serviceHasInstances(ctx context.Context, client CloudMapServicesAPI, serviceId *string) (bool, error) {
	paginator := servicediscovery.NewListInstancesPaginator(client, &servicediscovery.ListInstancesInput{
		ServiceId: serviceId,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return false, errors.WithStackTrace(err)
		}
		if len(page.Instances) > 0 {
			return true, nil
		}
	}

	return false, nil
}

// getCloudMapServiceTags retrieves all tags for a Cloud Map service.
func getCloudMapServiceTags(ctx context.Context, client CloudMapServicesAPI, serviceArn *string) (map[string]string, error) {
	output, err := client.ListTagsForResource(ctx, &servicediscovery.ListTagsForResourceInput{
		ResourceARN: serviceArn,
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	tags := make(map[string]string)
	for _, tag := range output.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return tags, nil
}

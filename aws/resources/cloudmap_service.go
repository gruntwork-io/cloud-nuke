package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAll retrieves all Cloud Map services in the current region that match the configured filters.
// It uses pagination to handle large numbers of services.
func (cms *CloudMapServices) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var result []*string
	
	// Create a paginator to iterate through all services
	paginator := servicediscovery.NewListServicesPaginator(cms.Client, &servicediscovery.ListServicesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(cms.Context)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		
		// Filter services based on configured rules (name patterns, creation time, etc.)
		for _, service := range page.Services {
			if configObj.CloudMapService.ShouldInclude(config.ResourceValue{
				Name: service.Name,
				Time: service.CreateDate,
			}) {
				result = append(result, service.Id)
			}
		}
	}
	
	return result, nil
}

// deregisterAllInstances removes all service instances registered with the given service.
// This is required before a service can be deleted.
func (cms *CloudMapServices) deregisterAllInstances(serviceId *string) error {
	// List all instances for this service
	paginator := servicediscovery.NewListInstancesPaginator(cms.Client, &servicediscovery.ListInstancesInput{
		ServiceId: serviceId,
	})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(cms.Context)
		if err != nil {
			return errors.WithStackTrace(err)
		}
		
		// Deregister each instance found
		for _, instance := range page.Instances {
			logging.Debugf("Deregistering instance %s from service %s", *instance.Id, *serviceId)
			
			_, err := cms.Client.DeregisterInstance(cms.Context, &servicediscovery.DeregisterInstanceInput{
				ServiceId:  serviceId,
				InstanceId: instance.Id,
			})
			
			// Log but don't fail on individual instance deregistration errors
			if err != nil {
				logging.Debugf("Error deregistering instance %s: %s", *instance.Id, err)
			}
		}
	}
	
	return nil
}

// waitForInstanceDeregistration waits for all instances to be fully deregistered from a service.
// Cloud Map requires all instances to be deregistered before a service can be deleted.
// This function polls for up to 2.5 minutes (30 retries * 5 seconds) for instances to be cleared.
func (cms *CloudMapServices) waitForInstanceDeregistration(serviceId *string) error {
	maxRetries := 30
	sleepBetweenRetries := 5 * time.Second
	
	for i := 0; i < maxRetries; i++ {
		// Check if any instances still exist for this service
		paginator := servicediscovery.NewListInstancesPaginator(cms.Client, &servicediscovery.ListInstancesInput{
			ServiceId: serviceId,
		})
		
		hasInstances := false
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(cms.Context)
			if err != nil {
				return errors.WithStackTrace(err)
			}
			
			if len(page.Instances) > 0 {
				hasInstances = true
				break
			}
		}
		
		// If no instances remain, service is safe to delete
		if !hasInstances {
			return nil
		}
		
		logging.Debugf("Waiting for instances in service %s to be deregistered (attempt %d/%d)", *serviceId, i+1, maxRetries)
		time.Sleep(sleepBetweenRetries)
	}
	
	return errors.WithStackTrace(fmt.Errorf("timeout waiting for instances to be deregistered in service %s", *serviceId))
}

// nukeAll deletes all specified Cloud Map services.
// It ensures that all service instances are deregistered before attempting to delete each service.
func (cms *CloudMapServices) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Cloud Map services to nuke in region %s", cms.Region)
		return nil
	}
	
	logging.Debugf("Deleting %d Cloud Map services in region %s", len(identifiers), cms.Region)
	
	var deletedServices []*string
	for _, id := range identifiers {
		// First, deregister all instances in the service
		err := cms.deregisterAllInstances(id)
		if err != nil {
			logging.Debugf("Error deregistering instances for service %s: %s", *id, err)
		}
		
		// Wait for all instances to be fully deregistered
		err = cms.waitForInstanceDeregistration(id)
		if err != nil {
			logging.Debugf("Error waiting for instances to be deregistered in service %s: %s", *id, err)
		}
		
		// Attempt to delete the service
		input := &servicediscovery.DeleteServiceInput{
			Id: id,
		}
		
		_, err = cms.Client.DeleteService(cms.Context, input)
		
		// Record the deletion attempt for reporting
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: "Cloud Map Service",
			Error:        err,
		}
		report.Record(e)
		
		if err != nil {
			logging.Debugf("Error deleting Cloud Map service %s: %s", *id, err)
		} else {
			logging.Debugf("Successfully deleted Cloud Map service: %s", *id)
			deletedServices = append(deletedServices, id)
		}
	}
	
	logging.Debugf("[OK] %d of %d Cloud Map service(s) deleted in %s", len(deletedServices), len(identifiers), cms.Region)
	
	return nil
}
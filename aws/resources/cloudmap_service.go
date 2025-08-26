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

func (cms *CloudMapServices) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var result []*string
	
	paginator := servicediscovery.NewListServicesPaginator(cms.Client, &servicediscovery.ListServicesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(cms.Context)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		
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

func (cms *CloudMapServices) deregisterAllInstances(serviceId *string) error {
	paginator := servicediscovery.NewListInstancesPaginator(cms.Client, &servicediscovery.ListInstancesInput{
		ServiceId: serviceId,
	})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(cms.Context)
		if err != nil {
			return errors.WithStackTrace(err)
		}
		
		for _, instance := range page.Instances {
			logging.Debugf("Deregistering instance %s from service %s", *instance.Id, *serviceId)
			
			_, err := cms.Client.DeregisterInstance(cms.Context, &servicediscovery.DeregisterInstanceInput{
				ServiceId:  serviceId,
				InstanceId: instance.Id,
			})
			
			if err != nil {
				logging.Debugf("Error deregistering instance %s: %s", *instance.Id, err)
			}
		}
	}
	
	return nil
}

func (cms *CloudMapServices) waitForInstanceDeregistration(serviceId *string) error {
	maxRetries := 30
	sleepBetweenRetries := 5 * time.Second
	
	for i := 0; i < maxRetries; i++ {
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
		
		if !hasInstances {
			return nil
		}
		
		logging.Debugf("Waiting for instances in service %s to be deregistered (attempt %d/%d)", *serviceId, i+1, maxRetries)
		time.Sleep(sleepBetweenRetries)
	}
	
	return errors.WithStackTrace(fmt.Errorf("timeout waiting for instances to be deregistered in service %s", *serviceId))
}

func (cms *CloudMapServices) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Cloud Map services to nuke in region %s", cms.Region)
		return nil
	}
	
	logging.Debugf("Deleting %d Cloud Map services in region %s", len(identifiers), cms.Region)
	
	var deletedServices []*string
	for _, id := range identifiers {
		err := cms.deregisterAllInstances(id)
		if err != nil {
			logging.Debugf("Error deregistering instances for service %s: %s", *id, err)
		}
		
		err = cms.waitForInstanceDeregistration(id)
		if err != nil {
			logging.Debugf("Error waiting for instances to be deregistered in service %s: %s", *id, err)
		}
		
		input := &servicediscovery.DeleteServiceInput{
			Id: id,
		}
		
		_, err = cms.Client.DeleteService(cms.Context, input)
		
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
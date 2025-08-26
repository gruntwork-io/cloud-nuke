package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (cns *CloudMapNamespaces) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var result []*string
	
	paginator := servicediscovery.NewListNamespacesPaginator(cns.Client, &servicediscovery.ListNamespacesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(cns.Context)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		
		for _, namespace := range page.Namespaces {
			if configObj.CloudMapNamespace.ShouldInclude(config.ResourceValue{
				Name: namespace.Name,
				Time: namespace.CreateDate,
			}) {
				result = append(result, namespace.Id)
			}
		}
	}
	
	return result, nil
}

func (cns *CloudMapNamespaces) waitForServicesToDelete(namespaceId *string) error {
	maxRetries := 30
	sleepBetweenRetries := 10 * time.Second
	
	for i := 0; i < maxRetries; i++ {
		paginator := servicediscovery.NewListServicesPaginator(cns.Client, &servicediscovery.ListServicesInput{
			Filters: []types.ServiceFilter{
				{
					Name:      types.ServiceFilterNameNamespaceId,
					Values:    []string{*namespaceId},
					Condition: types.FilterConditionEq,
				},
			},
		})
		
		hasServices := false
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(cns.Context)
			if err != nil {
				return errors.WithStackTrace(err)
			}
			
			if len(page.Services) > 0 {
				hasServices = true
				break
			}
		}
		
		if !hasServices {
			return nil
		}
		
		logging.Debugf("Waiting for services in namespace %s to be deleted (attempt %d/%d)", *namespaceId, i+1, maxRetries)
		time.Sleep(sleepBetweenRetries)
	}
	
	return errors.WithStackTrace(fmt.Errorf("timeout waiting for services to be deleted in namespace %s", *namespaceId))
}

func (cns *CloudMapNamespaces) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Cloud Map namespaces to nuke in region %s", cns.Region)
		return nil
	}
	
	logging.Debugf("Deleting %d Cloud Map namespaces in region %s", len(identifiers), cns.Region)
	
	var deletedNamespaces []*string
	for _, id := range identifiers {
		err := cns.waitForServicesToDelete(id)
		if err != nil {
			logging.Debugf("Error waiting for services to be deleted in namespace %s: %s", *id, err)
		}
		
		getResp, err := cns.Client.GetNamespace(cns.Context, &servicediscovery.GetNamespaceInput{
			Id: id,
		})
		if err != nil {
			logging.Debugf("Error getting namespace %s: %s", *id, err)
			continue
		}
		
		if getResp.Namespace.Properties != nil {
			if getResp.Namespace.Properties.DnsProperties != nil && getResp.Namespace.Properties.DnsProperties.HostedZoneId != nil {
				logging.Debugf("Namespace %s has associated Route53 hosted zone %s - it will be cleaned automatically", 
					*id, *getResp.Namespace.Properties.DnsProperties.HostedZoneId)
			}
		}
		
		input := &servicediscovery.DeleteNamespaceInput{
			Id: id,
		}
		
		_, err = cns.Client.DeleteNamespace(cns.Context, input)
		
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: "Cloud Map Namespace",
			Error:        err,
		}
		report.Record(e)
		
		if err != nil {
			logging.Debugf("Error deleting Cloud Map namespace %s: %s", *id, err)
		} else {
			logging.Debugf("Successfully deleted Cloud Map namespace: %s", *id)
			deletedNamespaces = append(deletedNamespaces, id)
		}
	}
	
	logging.Debugf("[OK] %d of %d Cloud Map namespace(s) deleted in %s", len(deletedNamespaces), len(identifiers), cns.Region)
	
	return nil
}
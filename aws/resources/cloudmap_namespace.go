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

// getAll retrieves all Cloud Map namespaces in the current region that match the configured filters.
// It uses pagination to handle large numbers of namespaces.
func (cns *CloudMapNamespaces) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var result []*string

	// Create a paginator to iterate through all namespaces
	paginator := servicediscovery.NewListNamespacesPaginator(cns.Client, &servicediscovery.ListNamespacesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(cns.Context)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Filter namespaces based on configured rules (name patterns, creation time, tags, etc.)
		for _, namespace := range page.Namespaces {
			// Get all tags for the namespace for filtering purposes
			tags, err := cns.getAllTags(namespace.Arn)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if configObj.CloudMapNamespace.ShouldInclude(config.ResourceValue{
				Name: namespace.Name,
				Time: namespace.CreateDate,
				Tags: tags,
			}) {
				result = append(result, namespace.Id)
			}
		}
	}

	return result, nil
}

// waitForServicesToDelete waits for all services within a namespace to be deleted.
// Cloud Map requires all services to be removed before a namespace can be deleted.
// This function polls for up to 5 minutes (30 retries * 10 seconds) for services to be cleared.
func (cns *CloudMapNamespaces) waitForServicesToDelete(namespaceId *string) error {
	maxRetries := 30
	sleepBetweenRetries := 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		// List all services that belong to this namespace
		paginator := servicediscovery.NewListServicesPaginator(cns.Client, &servicediscovery.ListServicesInput{
			Filters: []types.ServiceFilter{
				{
					Name:      types.ServiceFilterNameNamespaceId,
					Values:    []string{*namespaceId},
					Condition: types.FilterConditionEq,
				},
			},
		})

		// Check if any services still exist in the namespace
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

		// If no services remain, namespace is safe to delete
		if !hasServices {
			return nil
		}

		logging.Debugf("Waiting for services in namespace %s to be deleted (attempt %d/%d)", *namespaceId, i+1, maxRetries)
		time.Sleep(sleepBetweenRetries)
	}

	return errors.WithStackTrace(fmt.Errorf("timeout waiting for services to be deleted in namespace %s", *namespaceId))
}

// nukeAll deletes all specified Cloud Map namespaces.
// It ensures that services within each namespace are deleted first before attempting to delete the namespace.
func (cns *CloudMapNamespaces) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Cloud Map namespaces to nuke in region %s", cns.Region)
		return nil
	}

	logging.Debugf("Deleting %d Cloud Map namespaces in region %s", len(identifiers), cns.Region)

	var deletedNamespaces []*string
	for _, id := range identifiers {
		// First, wait for all services in the namespace to be deleted
		// This is required because Cloud Map won't allow namespace deletion with active services
		err := cns.waitForServicesToDelete(id)
		if err != nil {
			logging.Debugf("Error waiting for services to be deleted in namespace %s: %s", *id, err)
		}

		// Get namespace details to check for associated Route53 hosted zones
		getResp, err := cns.Client.GetNamespace(cns.Context, &servicediscovery.GetNamespaceInput{
			Id: id,
		})
		if err != nil {
			logging.Debugf("Error getting namespace %s: %s", *id, err)
			continue
		}

		// Log if namespace has DNS properties (Route53 hosted zone will be auto-cleaned)
		if getResp.Namespace.Properties != nil {
			if getResp.Namespace.Properties.DnsProperties != nil && getResp.Namespace.Properties.DnsProperties.HostedZoneId != nil {
				logging.Debugf("Namespace %s has associated Route53 hosted zone %s - it will be cleaned automatically",
					*id, *getResp.Namespace.Properties.DnsProperties.HostedZoneId)
			}
		}

		// Attempt to delete the namespace
		input := &servicediscovery.DeleteNamespaceInput{
			Id: id,
		}

		_, err = cns.Client.DeleteNamespace(cns.Context, input)

		// Record the deletion attempt for reporting
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

// getAllTags retrieves all tags for a given Cloud Map namespace.
// Returns a map of tag keys to tag values.
func (cns *CloudMapNamespaces) getAllTags(namespaceArn *string) (map[string]string, error) {
	input := &servicediscovery.ListTagsForResourceInput{
		ResourceARN: namespaceArn,
	}

	namespaceTags, err := cns.Client.ListTagsForResource(cns.Context, input)
	if err != nil {
		logging.Debugf("Error getting the tags for Cloud Map namespace with ARN %s", aws.ToString(namespaceArn))
		return nil, errors.WithStackTrace(err)
	}

	tags := make(map[string]string)
	for _, tag := range namespaceTags.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return tags, nil
}

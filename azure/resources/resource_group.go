package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// ResourceGroupsAPI is the part of the Azure Resource Manager API this resource
// uses. The SDK answers a list with a pager a test cannot build, so the paging
// and the long-running delete live in the adapter at the bottom of this file
// and the decisions stay testable.
type ResourceGroupsAPI interface {
	// ListResourceGroups returns every resource group in the subscription.
	ListResourceGroups(ctx context.Context) ([]ResourceGroup, error)

	// DeleteResourceGroup deletes one group and waits for the operation to
	// finish. Deleting one that is already gone is not an error. Azure deletes
	// everything inside a group with it, which is why this is the one resource
	// type that cleans up a whole test's worth of infrastructure.
	DeleteResourceGroup(ctx context.Context, name string) error
}

// ResourceGroup is what this resource needs about one group.
type ResourceGroup struct {
	Name     string
	Location string
	Tags     map[string]string

	// Created is when the group was created. Azure reports it only when the
	// caller asks for the createdTime field, and it is the zero time when the
	// subscription does not expose it.
	Created time.Time

	// Locked is true when the group carries a delete lock, which makes a
	// delete fail, so cloud-nuke reports it rather than attempting it.
	Locked bool

	// Managed is true when another service owns the group's lifecycle, such as
	// an AKS node group. Deleting one of those breaks the service that made it.
	Managed bool
}

// NewResourceGroups creates the Azure resource group resource.
func NewResourceGroups() AzureResource {
	adapter := &AzureResourceAdapter[ResourceGroupsAPI]{locationScope: LocationReportedByLister}

	adapter.Resource = &resource.Resource[ResourceGroupsAPI]{
		ResourceTypeName: "resource-group",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAzureInitClient(func(r *resource.Resource[ResourceGroupsAPI], cfg AzureConfig) {
			credential, err := azidentity.NewDefaultAzureCredential(nil)
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to read Azure credentials: %w", err)
				return
			}

			client, err := armresources.NewResourceGroupsClient(cfg.SubscriptionID, credential, nil)
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Azure resource groups client: %w", err)
				return
			}

			r.Scope.Subscription = cfg.SubscriptionID
			r.Client = resourceGroupsClient{client: client}
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.AzureResourceGroup
		},
		Lister: func(ctx context.Context, client ResourceGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return listResourceGroups(ctx, client, cfg, adapter.Config())
		},
		Nuker: resource.SequentialDeleter(deleteResourceGroup),
	}

	return adapter
}

// listResourceGroups returns the resource groups in the subscription that match
// the config filters and the location filter.
func listResourceGroups(ctx context.Context, client ResourceGroupsAPI, cfg config.ResourceType, azureCfg AzureConfig) ([]*string, error) {
	groups, err := client.ListResourceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing resource groups: %w", err)
	}

	var result []*string
	for _, group := range groups {
		// A group another service owns goes when that service goes. Deleting an
		// AKS node group by hand leaves the cluster broken.
		if group.Managed {
			logging.Debugf("Skipping resource group %s: another service manages it", group.Name)
			continue
		}

		// A delete lock makes the API refuse the delete, so saying so during
		// the scan is better than a failure per group during the nuke.
		if group.Locked {
			logging.Debugf("Skipping resource group %s: it carries a delete lock", group.Name)
			continue
		}

		if !azureCfg.MatchesLocationFilter(group.Location) {
			continue
		}

		name := group.Name
		created := group.Created

		value := config.ResourceValue{Name: &name, Tags: group.Tags}
		if !created.IsZero() {
			value.Time = &created
		}

		if cfg.ShouldInclude(value) {
			result = append(result, &name)
		}
	}

	return result, nil
}

// deleteResourceGroup deletes one resource group, and everything in it.
func deleteResourceGroup(ctx context.Context, client ResourceGroupsAPI, name *string) error {
	if err := client.DeleteResourceGroup(ctx, *name); err != nil {
		return fmt.Errorf("error deleting resource group %s: %w", *name, err)
	}

	logging.Debugf("Deleted resource group: %s", *name)
	return nil
}

// resourceGroupsClient drives the SDK: the pager, the long-running delete and
// the fields Azure only reports when asked.
type resourceGroupsClient struct {
	client *armresources.ResourceGroupsClient
}

func (c resourceGroupsClient) ListResourceGroups(ctx context.Context) ([]ResourceGroup, error) {
	var found []ResourceGroup

	// createdTime is not in the default response, so it is asked for by name.
	pager := c.client.NewListPager(&armresources.ResourceGroupsClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, group := range page.Value {
			if group == nil || group.Name == nil {
				continue
			}

			resourceGroup := ResourceGroup{
				Name: *group.Name,
				Tags: map[string]string{},
			}
			if group.Location != nil {
				resourceGroup.Location = *group.Location
			}
			for key, value := range group.Tags {
				if value != nil {
					resourceGroup.Tags[key] = *value
				}
			}
			if group.ManagedBy != nil && strings.TrimSpace(*group.ManagedBy) != "" {
				resourceGroup.Managed = true
			}

			found = append(found, resourceGroup)
		}
	}

	return found, nil
}

func (c resourceGroupsClient) DeleteResourceGroup(ctx context.Context, name string) error {
	poller, err := c.client.BeginDelete(ctx, name, nil)
	if err != nil {
		return err
	}

	// Azure deletes a group's contents before the group, so this waits rather
	// than returning while the delete is still running: a later resource type
	// could otherwise be deleted out from under it.
	if _, err := poller.PollUntilDone(ctx, nil); err != nil {
		return fmt.Errorf("waiting for deletion: %w", err)
	}

	return nil
}

var _ ResourceGroupsAPI = resourceGroupsClient{}

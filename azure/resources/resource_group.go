package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armlocks"
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

	// Locked is true when a lock at or under the group would make the delete
	// fail, so the scan skips the group rather than reporting a group it
	// cannot delete.
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

			locks, err := armlocks.NewManagementLocksClient(cfg.SubscriptionID, credential, nil)
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Azure management locks client: %w", err)
				return
			}

			r.Scope.Subscription = cfg.SubscriptionID
			r.Client = resourceGroupsClient{client: client, locks: locks}
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

	// Azure reports no creation time for a resource group, so a run with a time
	// filter excludes every one of them for safety, as it does for any resource
	// whose age it cannot read. That is the right call and a silent one, so the
	// run says how many it left alone and why.
	ageUnknown := 0
	lockedOrManaged := 0

	var result []*string
	for _, group := range groups {
		// A group another service owns goes when that service goes. Deleting an
		// AKS node group by hand leaves the cluster broken.
		if group.Managed {
			logging.Debugf("Skipping resource group %s: another service manages it", group.Name)
			lockedOrManaged++
			continue
		}

		// A delete lock makes the API refuse the delete, so saying so during
		// the scan is better than a failure per group during the nuke.
		if group.Locked {
			logging.Debugf("Skipping resource group %s: it carries a delete lock", group.Name)
			lockedOrManaged++
			continue
		}

		if !azureCfg.MatchesLocationFilter(group.Location) {
			continue
		}

		// Azure reports no age for a resource group, so a run that filters on
		// age can delete none of them. The config layer already excludes a
		// resource whose age is unknown; this says how many, since a sweep
		// that silently finds nothing reads as a clean subscription.
		if cfg.HasTimeFilter() {
			ageUnknown++
			continue
		}

		name := group.Name
		value := config.ResourceValue{Name: &name, Tags: group.Tags}

		if cfg.ShouldInclude(value) {
			result = append(result, &name)
		}
	}

	if lockedOrManaged > 0 {
		logging.Warnf(
			"Leaving %d resource groups alone: a lock, on the group or on the subscription, or another service that owns them. Azure refuses those deletes.",
			lockedOrManaged)
	}

	if len(result) > 0 && azureCfg.HasLocationFilter() {
		logging.Warnf(
			"A resource group's delete takes everything inside it, wherever that lives, so this run can remove resources in locations it was not narrowed to.")
	}

	if ageUnknown > 0 {
		logging.Warnf(
			"Leaving %d resource groups alone: Azure does not report when a resource group was created, and a time filter cannot be applied to an age it cannot read. Drop --older-than and --newer-than to include them.",
			ageUnknown)
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
	locks  *armlocks.ManagementLocksClient
}

func (c resourceGroupsClient) ListResourceGroups(ctx context.Context) ([]ResourceGroup, error) {
	var found []ResourceGroup

	locked, subscriptionLocked := c.lockedGroups(ctx)

	// The list options carry only a filter and a page size. Azure reports no
	// creation time for a resource group at all: the created time the resources
	// API can expand belongs to the resources inside a group, not to the group,
	// which is why this type carries no age and the lister says so when a run
	// filters on one.
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
			resourceGroup.Locked = subscriptionLocked || locked[strings.ToLower(resourceGroup.Name)]

			found = append(found, resourceGroup)
		}
	}

	return found, nil
}

// lockedGroups returns the lower-cased names of the resource groups a lock
// would stop cloud-nuke from deleting.
//
// One subscription-level call answers for every scope, and a lock anywhere at
// or under a group counts: deleting a group deletes its contents, so a
// CanNotDelete lock on a single resource inside it fails the whole delete.
//
// Reading locks needs Microsoft.Authorization/locks/read, which a credential
// that can otherwise nuke may not have. A failure there is not fatal: the scan
// goes on with nothing marked locked, and a locked group then reports the
// delete error it would have reported before.
func (c resourceGroupsClient) lockedGroups(ctx context.Context) (map[string]bool, bool) {
	locked := map[string]bool{}
	subscriptionLocked := false
	if c.locks == nil {
		return locked, false
	}

	pager := c.locks.NewListAtSubscriptionLevelPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logging.Warnf("Could not read Azure management locks, so no resource group is treated as locked and a locked one will fail its delete: %v", err)
			return map[string]bool{}, false
		}

		for _, lock := range page.Value {
			if lock == nil || lock.ID == nil || lock.Properties == nil {
				continue
			}

			// NotSpecified blocks nothing. ReadOnly blocks a delete the same
			// way CanNotDelete does.
			level := lock.Properties.Level
			if level == nil || (*level != armlocks.LockLevelCanNotDelete && *level != armlocks.LockLevelReadOnly) {
				continue
			}

			if group := resourceGroupFromID(*lock.ID); group != "" {
				locked[group] = true
			} else if isSubscriptionLockID(*lock.ID) {
				// A lock at the subscription covers every group in it.
				subscriptionLocked = true
			}
		}
	}

	return locked, subscriptionLocked
}

// isSubscriptionLockID reports whether an ID names a lock on the subscription
// itself, which is exactly /subscriptions/<id>/providers/Microsoft.Authorization/locks/<name>.
// It matches that shape rather than the prefix, so an ID shape this code does
// not know cannot mark every group in the subscription locked.
func isSubscriptionLockID(id string) bool {
	parts := strings.Split(strings.TrimPrefix(id, "/"), "/")
	return len(parts) == 6 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.Authorization") &&
		strings.EqualFold(parts[4], "locks")
}

// resourceGroupFromID reads the resource group out of an Azure resource ID such
// as /subscriptions/s/resourceGroups/g/providers/Microsoft.Authorization/locks/l,
// lower-cased because Azure echoes the casing back as it was typed. It returns
// an empty string for a subscription-scoped ID, which names no group.
func resourceGroupFromID(id string) string {
	parts := strings.Split(id, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "resourceGroups") && i+1 < len(parts) {
			return strings.ToLower(parts[i+1])
		}
	}
	return ""
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

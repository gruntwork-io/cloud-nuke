package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"google.golang.org/api/googleapi"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

// NewCloudSQLInstances creates a new Cloud SQL instance resource using the generic resource pattern.
func NewCloudSQLInstances() GcpResource {
	return NewGcpResource(&resource.Resource[*sqladmin.Service]{
		ResourceTypeName: "cloud-sql-instance",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*sqladmin.Service], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := sqladmin.NewService(context.Background())
			if err != nil {
				panic(fmt.Sprintf("failed to create Cloud SQL client: %v", err))
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GcpCloudSQLInstance
		},
		Lister: listCloudSQLInstances,
		Nuker:  resource.SequentialDeleter(deleteCloudSQLInstance),
	})
}

// instanceSkipStates contains Cloud SQL instance states in which deletion is not possible.
// RUNNABLE, STOPPED, FAILED, and SUSPENDED instances can all be deleted.
var instanceSkipStates = map[string]bool{
	"PENDING_CREATE":     true, // creation in progress — API rejects delete
	"PENDING_DELETE":     true, // already being deleted
	"MAINTENANCE":        true, // instance is offline for maintenance
	"ONLINE_MAINTENANCE": true, // deprecated, but guard against it
	"REPAIRING":          true, // read pool node being repaired — not safe to delete
}

// listCloudSQLInstances retrieves all Cloud SQL instances in the project that match the config filters.
//
// Read replicas and read pool instances are returned before primary instances to ensure
// they are deleted first — the API rejects deletion of a primary that still has replicas.
func listCloudSQLInstances(ctx context.Context, client *sqladmin.Service, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var replicas []*string
	var primaries []*string

	err := client.Instances.List(scope.ProjectID).Context(ctx).Pages(ctx, func(page *sqladmin.InstancesListResponse) error {
		for _, instance := range page.Items {
			// Skip instances not managed by Cloud SQL (external or on-premises).
			if instance.BackendType == "EXTERNAL" || instance.InstanceType == "ON_PREMISES_INSTANCE" {
				logging.Debugf("Skipping externally managed Cloud SQL instance: %s", instance.Name)
				continue
			}

			// Skip instances in states where deletion is not currently possible.
			if instanceSkipStates[instance.State] {
				logging.Warnf("Skipping Cloud SQL instance %s: instance is in state %s", instance.Name, instance.State)
				continue
			}

			// Skip instances with deletion protection enabled.
			if instance.Settings != nil && instance.Settings.DeletionProtectionEnabled {
				logging.Warnf("Skipping Cloud SQL instance %s: deletion protection is enabled", instance.Name)
				continue
			}

			createdAt, err := time.Parse(time.RFC3339, instance.CreateTime)
			if err != nil {
				logging.Warnf("Skipping Cloud SQL instance %s: failed to parse creation timestamp: %v", instance.Name, err)
				continue
			}

			var labels map[string]string
			if instance.Settings != nil {
				labels = instance.Settings.UserLabels
			}
			if labels == nil {
				labels = map[string]string{}
			}

			resourceValue := config.ResourceValue{
				Name: &instance.Name,
				Time: &createdAt,
				Tags: labels,
			}

			if cfg.ShouldInclude(resourceValue) {
				id := fmt.Sprintf("%s/%s", scope.ProjectID, instance.Name)
				// Replicas and read pool nodes must be deleted before their primary.
				if instance.InstanceType == "READ_REPLICA_INSTANCE" || instance.InstanceType == "READ_POOL_INSTANCE" {
					replicas = append(replicas, &id)
				} else {
					primaries = append(primaries, &id)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error listing Cloud SQL instances: %w", err)
	}

	// Replicas are listed first to ensure correct deletion order.
	return append(replicas, primaries...), nil
}

// deleteCloudSQLInstance deletes a single Cloud SQL instance and waits for the operation to complete.
func deleteCloudSQLInstance(ctx context.Context, client *sqladmin.Service, id *string) error {
	project, name, err := parseCloudSQLInstanceID(*id)
	if err != nil {
		return err
	}

	op, err := client.Instances.Delete(project, name).Context(ctx).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			logging.Debugf("Cloud SQL instance %s already deleted, skipping", *id)
			return nil
		}
		return fmt.Errorf("error deleting Cloud SQL instance %s: %w", *id, err)
	}

	if err := waitForCloudSQLOperation(ctx, client, project, op.Name); err != nil {
		return fmt.Errorf("error waiting for deletion of Cloud SQL instance %s: %w", *id, err)
	}

	logging.Debugf("Deleted Cloud SQL instance: %s", *id)
	return nil
}

// waitForCloudSQLOperation polls a Cloud SQL long-running operation until it completes
// or the context is cancelled.
func waitForCloudSQLOperation(ctx context.Context, client *sqladmin.Service, project, opName string) error {
	for {
		op, err := client.Operations.Get(project, opName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("error polling Cloud SQL operation %s: %w", opName, err)
		}

		if op.Status == "DONE" {
			if op.Error != nil && len(op.Error.Errors) > 0 {
				return fmt.Errorf("cloud SQL operation %s failed: %s", opName, op.Error.Errors[0].Message)
			}
			return nil
		}

		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// parseCloudSQLInstanceID parses a composite ID of the form "project/instance".
func parseCloudSQLInstanceID(id string) (project, name string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid Cloud SQL instance ID %q: expected format project/instance", id)
	}
	return parts[0], parts[1], nil
}

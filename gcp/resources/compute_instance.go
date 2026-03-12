package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	compute "google.golang.org/api/compute/v1"
)

// NewComputeInstances creates a new Compute Engine VM instance resource using the generic resource pattern.
func NewComputeInstances() GcpResource {
	return NewGcpResource(&resource.Resource[*compute.Service]{
		ResourceTypeName: "compute-instance",
		BatchSize:        ComputeInstanceBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*compute.Service], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := compute.NewService(context.Background())
			if err != nil {
				panic(fmt.Sprintf("failed to create Compute Engine client: %v", err))
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ComputeInstance
		},
		Lister: listComputeInstances,
		Nuker:  resource.SimpleBatchDeleter(deleteComputeInstance),
	})
}

// ComputeInstanceBatchSize controls the number of instances per nuke batch.
// The effective concurrency is capped at resource.DefaultMaxConcurrent by the
// semaphore in SimpleBatchDeleter; this batch size just controls how many
// identifiers are handed to a single Nuke call.
const ComputeInstanceBatchSize = 20

// computeDeleteDelay is the rate-limiting delay between instance deletions to
// avoid GCP Compute Engine API quota issues (default 20 req/s per project).
const computeDeleteDelay = 500 * time.Millisecond

// listComputeInstances retrieves all Compute Engine VM instances across all zones in the project.
func listComputeInstances(ctx context.Context, client *compute.Service, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	err := client.Instances.AggregatedList(scope.ProjectID).Pages(ctx, func(list *compute.InstanceAggregatedList) error {
		for zonePath, scopedList := range list.Items {
			// Only process zone-scoped entries (skip regions/ or other scopes)
			if !strings.HasPrefix(zonePath, "zones/") {
				continue
			}
			if scopedList.Instances == nil {
				continue
			}
			zone := extractZoneName(zonePath)
			for _, instance := range scopedList.Instances {
				// Skip terminated instances -- they are already gone and
				// delete attempts would return 404.
				if instance.Status == "TERMINATED" {
					logging.Debugf("Skipping terminated instance %s in zone %s", instance.Name, zone)
					continue
				}

				// Skip suspended instances -- GCP returns 400 on delete;
				// the instance must be resumed or stopped first.
				if instance.Status == "SUSPENDED" {
					logging.Warnf("Skipping instance %s in zone %s: instance is suspended (must be resumed before deletion)", instance.Name, zone)
					continue
				}

				// Skip instances with deletion protection enabled -- the
				// delete API would return 400 with no actionable recourse.
				if instance.DeletionProtection {
					logging.Warnf("Skipping instance %s in zone %s: deletion protection is enabled", instance.Name, zone)
					continue
				}

				createdAt, err := time.Parse(time.RFC3339Nano, instance.CreationTimestamp)
				if err != nil {
					logging.Warnf("Skipping instance %s: failed to parse creation timestamp: %v", instance.Name, err)
					continue
				}

				tags := make(map[string]string, len(instance.Labels))
				for k, v := range instance.Labels {
					tags[k] = v
				}

				resourceValue := config.ResourceValue{
					Name: &instance.Name,
					Time: &createdAt,
					Tags: tags,
				}

				if cfg.ShouldInclude(resourceValue) {
					id := fmt.Sprintf("%s/%s/%s", scope.ProjectID, zone, instance.Name)
					result = append(result, &id)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error listing compute instances: %w", err)
	}

	return result, nil
}

// deleteComputeInstance deletes a single Compute Engine VM instance and waits
// for the operation to complete.
func deleteComputeInstance(ctx context.Context, client *compute.Service, id *string) error {
	project, zone, name, err := parseComputeInstanceID(*id)
	if err != nil {
		return err
	}

	op, err := client.Instances.Delete(project, zone, name).Context(ctx).Do()
	if err != nil {
		if isGCPNotFound(err) {
			logging.Debugf("Compute instance %s already deleted, skipping", *id)
			return nil
		}
		return fmt.Errorf("error deleting compute instance %s: %w", *id, err)
	}

	// Wait for the delete operation to complete
	if err := waitForZoneOperation(ctx, client, project, zone, op.Name); err != nil {
		return fmt.Errorf("error waiting for deletion of compute instance %s: %w", *id, err)
	}

	logging.Debugf("Deleted compute instance: %s", *id)

	// Rate-limiting delay to avoid GCP API quota issues
	time.Sleep(computeDeleteDelay)

	return nil
}

// parseComputeInstanceID parses a composite ID of the form "project/zone/name".
func parseComputeInstanceID(id string) (project, zone, name string, err error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("invalid compute instance ID %q: expected format project/zone/name", id)
	}
	return parts[0], parts[1], parts[2], nil
}

// extractZoneName extracts the zone name from an aggregated list key (e.g., "zones/us-central1-a" -> "us-central1-a").
func extractZoneName(zonePath string) string {
	return strings.TrimPrefix(zonePath, "zones/")
}

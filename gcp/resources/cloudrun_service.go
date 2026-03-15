package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	run "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewCloudRunServices creates a new Cloud Run service resource using the generic resource pattern.
func NewCloudRunServices() GcpResource {
	return NewGcpResource(&resource.Resource[*run.ServicesClient]{
		ResourceTypeName: "cloud-run-service",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*run.ServicesClient], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := run.NewServicesClient(context.Background())
			if err != nil {
				panic(fmt.Sprintf("failed to create Cloud Run services client: %v", err))
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GcpCloudRunService
		},
		Lister: listCloudRunServices,
		Nuker:  resource.SequentialDeleter(deleteCloudRunService),
	})
}

// listCloudRunServices retrieves all Cloud Run services across all regions in the project
// that match the config filters. Locations are enumerated via listCloudRunLocations and
// queried individually — the Cloud Run API does not support the "locations/-" wildcard.
func listCloudRunServices(ctx context.Context, client *run.ServicesClient, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	locations, err := listCloudRunLocations(ctx, scope.ProjectID)
	if err != nil {
		return nil, err
	}

	var result []*string

	for _, location := range locations {
		parent := fmt.Sprintf("projects/%s/locations/%s", scope.ProjectID, location)

		it := client.ListServices(ctx, &runpb.ListServicesRequest{Parent: parent})
		for {
			svc, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				// Some locations may not have Cloud Run enabled — skip them rather than
				// aborting the entire listing.
				logging.Debugf("error listing Cloud Run services in %s, skipping location: %v", location, err)
				break
			}

			// Extract the short name for config filtering; the full resource name is
			// retained as the delete identifier since the API requires it.
			shortName := svc.Name[strings.LastIndex(svc.Name, "/")+1:]

			var resourceTime time.Time
			if svc.GetCreateTime() != nil {
				resourceTime = svc.GetCreateTime().AsTime()
			}

			labels := svc.GetLabels()
			if labels == nil {
				labels = map[string]string{}
			}

			resourceValue := config.ResourceValue{
				Name: &shortName,
				Time: &resourceTime,
				Tags: labels,
			}

			if cfg.ShouldInclude(resourceValue) {
				name := svc.Name
				result = append(result, &name)
			}
		}
	}

	return result, nil
}

// deleteCloudRunService deletes a single Cloud Run service and waits for the operation to complete.
func deleteCloudRunService(ctx context.Context, client *run.ServicesClient, name *string) error {
	serviceName := *name

	op, err := client.DeleteService(ctx, &runpb.DeleteServiceRequest{Name: serviceName})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Cloud Run service %s already deleted, skipping", serviceName)
			return nil
		}
		return fmt.Errorf("error deleting Cloud Run service %s: %w", serviceName, err)
	}

	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("error waiting for Cloud Run service %s deletion: %w", serviceName, err)
	}

	logging.Debugf("Deleted Cloud Run service: %s", serviceName)
	return nil
}

package resources

import (
	"context"
	"fmt"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	cloudinv "github.com/james00012/cloud-inventory"
	gcpinv "github.com/james00012/cloud-inventory/gcp"
)

// CloudRunServicesAPI is what this resource needs of the shared inventory: the
// list and the delete. The library owns the API versions, the fan-out over
// locations and the long-running delete; this file owns which services the run
// deletes.
type CloudRunServicesAPI interface {
	cloudinv.Lister[gcpinv.CloudRunService, gcpinv.Scope]
	cloudinv.Deleter
}

// NewCloudRunServices creates the Cloud Run service resource.
func NewCloudRunServices() GcpResource {
	gcpCfg := &GcpConfig{}

	return NewGcpResourceWithLocationScope(&resource.Resource[CloudRunServicesAPI]{
		ResourceTypeName: "cloud-run-service",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[CloudRunServicesAPI], cfg GcpConfig) {
			*gcpCfg = cfg
			r.Scope.ProjectID = cfg.ProjectID

			client, err := gcpinv.NewCloudRunServices(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Cloud Run client: %w", err)
				return
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GcpCloudRunService
		},
		Lister: func(ctx context.Context, client CloudRunServicesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return listCloudRunServices(ctx, client, scope, cfg, *gcpCfg)
		},
		Nuker: resource.SequentialDeleter(deleteCloudRunService),
	}, LocationAppliedByLister)
}

// listCloudRunServices asks the inventory for the regions this run allows, and
// keeps the services the config filters keep.
func listCloudRunServices(ctx context.Context, client CloudRunServicesAPI, scope resource.Scope, cfg config.ResourceType, gcpCfg GcpConfig) ([]*string, error) {
	// Both halves of --region go to the library, so a region this run rules out
	// is never asked about and never counted among the ones it depends on.
	services, incomplete, err := client.List(ctx, gcpinv.Scope{
		ProjectID:        scope.ProjectID,
		Locations:        gcpCfg.Locations,
		ExcludeLocations: gcpCfg.ExcludeLocations,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing Cloud Run services: %w", err)
	}

	// Every region failing is not a project with no services: it is a run that
	// learned nothing, and reporting zero would call the project clean. The
	// cause is wrapped so the provider's own message, and the classification
	// the provider layer reads from it, both survive.
	if incomplete.Total() {
		if cause := incomplete.Cause(); cause != nil {
			return nil, fmt.Errorf("could not list Cloud Run services in any of the %d regions asked for (%s): %w",
				incomplete.Asked, incomplete.String(), cause)
		}
		return nil, fmt.Errorf("could not list Cloud Run services in any of the %d regions asked for: %s",
			incomplete.Asked, incomplete.String())
	}
	if incomplete.Any() {
		logging.Warnf("Could not list Cloud Run services in %d of %d regions (%s), so any service there is left alone.",
			len(incomplete.Gaps), incomplete.Asked, incomplete.String())
	}

	var result []*string
	for _, service := range services {
		identifier := string(service.ID)
		name := service.DisplayName()
		created := service.CreatedAt()

		value := config.ResourceValue{Name: &name, Tags: service.Tags()}
		if !created.IsZero() {
			value.Time = &created
		}

		if cfg.ShouldInclude(value) {
			result = append(result, &identifier)
		}
	}

	return result, nil
}

// deleteCloudRunService deletes one service by its full name.
func deleteCloudRunService(ctx context.Context, client CloudRunServicesAPI, name *string) error {
	if err := client.Delete(ctx, cloudinv.ID(*name)); err != nil {
		return fmt.Errorf("error deleting Cloud Run service %s: %w", *name, err)
	}

	logging.Debugf("Deleted Cloud Run service: %s", *name)
	return nil
}

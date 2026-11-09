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
	// Naming the locations is how a run narrowed by --region costs one call
	// rather than one per region Cloud Run offers.
	services, incomplete, err := client.List(ctx, gcpinv.Scope{
		ProjectID: scope.ProjectID,
		Locations: gcpCfg.Locations,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing Cloud Run services: %w", err)
	}

	// Every region failing is not a project with no services: it is a run that
	// learned nothing, and reporting zero would call the project clean.
	if incomplete.Total() {
		return nil, fmt.Errorf("could not list Cloud Run services in any of the %d regions asked for: %s",
			incomplete.Asked, incomplete.String())
	}
	if incomplete.Any() {
		logging.Warnf("Could not list Cloud Run services in %s, so any service there is left alone.", incomplete.String())
	}

	var result []*string
	for _, service := range services {
		// An exclusion is read here, because the library takes the regions to
		// ask rather than the rule that produced them.
		if !gcpCfg.MatchesLocationFilter(service.Location) {
			continue
		}

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

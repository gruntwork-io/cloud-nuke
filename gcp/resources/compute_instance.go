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
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// ComputeInstancesAPI is the part of the Compute Engine API this resource uses.
// The SDK answers a list with a paged aggregate a test cannot build, so the
// paging and the long-running delete live in the adapter at the bottom of this
// file and the decisions stay testable.
type ComputeInstancesAPI interface {
	// ListInstances returns every instance in the project, in every zone, from
	// the one aggregated call the API offers.
	ListInstances(ctx context.Context, projectID string) ([]ComputeInstance, error)

	// DeleteInstance deletes one instance and waits for the operation to
	// finish. Deleting one that is already gone is not an error.
	DeleteInstance(ctx context.Context, projectID, zone, name string) error
}

// ComputeInstance is what this resource needs about one virtual machine.
type ComputeInstance struct {
	Name string

	// Zone is where the instance runs, such as us-central1-a. A zone is not a
	// location of the kind a resource name carries, which is why this type
	// filters inside its own lister.
	Zone string

	Created time.Time
	Labels  map[string]string

	// Protected is true when the instance carries deletion protection, which
	// makes the delete fail until somebody turns it off.
	Protected bool
}

// NewComputeInstances creates the Compute Engine instance resource.
func NewComputeInstances() GcpResource {
	// gcpCfg is filled in by InitClient and read by the lister, which is how a
	// resource sees the location filter: the generic Lister signature carries
	// the resource's own config, not the provider's.
	gcpCfg := &GcpConfig{}

	// An instance's identifier carries its zone rather than a location, and a
	// zone is not what --region names, so this lister applies the filter.
	return NewGcpResourceWithLocationScope(&resource.Resource[ComputeInstancesAPI]{
		ResourceTypeName: "compute-instance",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[ComputeInstancesAPI], cfg GcpConfig) {
			*gcpCfg = cfg
			r.Scope.ProjectID = cfg.ProjectID

			service, err := compute.NewService(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Compute Engine client: %w", err)
				return
			}
			r.Client = computeInstancesClient{service: service}
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ComputeInstance
		},
		Lister: func(ctx context.Context, client ComputeInstancesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return listComputeInstances(ctx, client, scope, cfg, *gcpCfg)
		},
		Nuker: resource.SequentialDeleter(deleteComputeInstance),
	}, LocationAppliedByLister)
}

// listComputeInstances returns the instances in the project that match the
// config filters and the location filter.
func listComputeInstances(ctx context.Context, client ComputeInstancesAPI, scope resource.Scope, cfg config.ResourceType, gcpCfg GcpConfig) ([]*string, error) {
	instances, err := client.ListInstances(ctx, scope.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("error listing compute instances: %w", err)
	}

	protected := 0

	var result []*string
	for _, instance := range instances {
		if !zoneMatchesLocationFilter(gcpCfg, instance.Zone) {
			continue
		}

		// A protected instance refuses its delete until somebody turns the
		// protection off, so the run says it is there rather than reporting a
		// failure per instance during the nuke.
		if instance.Protected {
			logging.Debugf("Skipping compute instance %s: it carries deletion protection", instance.Name)
			protected++
			continue
		}

		name := instance.Name
		created := instance.Created

		value := config.ResourceValue{Name: &name, Tags: instance.Labels}
		if !created.IsZero() {
			value.Time = &created
		}

		if cfg.ShouldInclude(value) {
			identifier := instanceIdentifier(scope.ProjectID, instance.Zone, instance.Name)
			result = append(result, &identifier)
		}
	}

	if protected > 0 {
		logging.Warnf("Leaving %d compute instances alone: they carry deletion protection, which Compute Engine refuses to delete through.", protected)
	}

	return result, nil
}

// zoneMatchesLocationFilter reports whether an instance in this zone survives
// the --region filter. An engineer who writes --region us-central1 means every
// zone in it, and one who writes a zone means that zone, so both are matched,
// and an exclusion of either beats an inclusion of the other.
func zoneMatchesLocationFilter(cfg GcpConfig, zone string) bool {
	if !cfg.HasLocationFilter() {
		return true
	}

	region := regionOfZone(zone)
	for _, excluded := range cfg.ExcludeLocations {
		if strings.EqualFold(excluded, zone) || strings.EqualFold(excluded, region) {
			return false
		}
	}

	if len(cfg.Locations) == 0 {
		return true
	}

	for _, wanted := range cfg.Locations {
		if strings.EqualFold(wanted, zone) || strings.EqualFold(wanted, region) {
			return true
		}
	}

	return false
}

// regionOfZone reads the region out of a zone name: us-central1-a is in
// us-central1. A value that is not a zone is returned unchanged, so a filter
// written as a region still matches itself.
func regionOfZone(zone string) string {
	if cut := strings.LastIndex(zone, "-"); cut > 0 {
		return zone[:cut]
	}
	return zone
}

// instanceIdentifier is the full resource path, which is what the delete needs
// and what the report shows, since an instance name is unique only in its zone.
func instanceIdentifier(projectID, zone, name string) string {
	return fmt.Sprintf("projects/%s/zones/%s/instances/%s", projectID, zone, name)
}

// deleteComputeInstance deletes one instance, read back out of its identifier.
func deleteComputeInstance(ctx context.Context, client ComputeInstancesAPI, identifier *string) error {
	projectID, zone, name, err := parseInstanceIdentifier(*identifier)
	if err != nil {
		return err
	}

	if err := client.DeleteInstance(ctx, projectID, zone, name); err != nil {
		return fmt.Errorf("error deleting compute instance %s: %w", *identifier, err)
	}

	logging.Debugf("Deleted compute instance: %s", *identifier)
	return nil
}

// parseInstanceIdentifier reads the project, the zone and the name back out of
// what the lister returned. An identifier of another shape is a programming
// error rather than something the user typed, so it fails loudly.
func parseInstanceIdentifier(identifier string) (projectID, zone, name string, err error) {
	parts := strings.Split(identifier, "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "zones" || parts[4] != "instances" {
		return "", "", "", fmt.Errorf("%q is not a compute instance path of the form projects/<project>/zones/<zone>/instances/<name>", identifier)
	}
	return parts[1], parts[3], parts[5], nil
}

// computeInstancesClient drives the SDK: the aggregated list, which is the one
// call that covers every zone, and the long-running delete.
type computeInstancesClient struct {
	service *compute.Service
}

func (c computeInstancesClient) ListInstances(ctx context.Context, projectID string) ([]ComputeInstance, error) {
	var found []ComputeInstance

	err := c.service.Instances.AggregatedList(projectID).Pages(ctx, func(page *compute.InstanceAggregatedList) error {
		for scope, scoped := range page.Items {
			// The aggregate carries a row per scope, including regions and
			// scopes holding nothing but a warning.
			if !strings.HasPrefix(scope, "zones/") {
				continue
			}

			for _, instance := range scoped.Instances {
				if instance == nil {
					continue
				}

				created, err := time.Parse(time.RFC3339, instance.CreationTimestamp)
				if err != nil {
					// An instance whose age cannot be read is still an
					// instance; the config layer refuses it under a time
					// filter rather than this dropping it here.
					logging.Debugf("Could not read the creation time of compute instance %s: %v", instance.Name, err)
					created = time.Time{}
				}

				found = append(found, ComputeInstance{
					Name:      instance.Name,
					Zone:      strings.TrimPrefix(scope, "zones/"),
					Created:   created,
					Labels:    instance.Labels,
					Protected: instance.DeletionProtection,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return found, nil
}

func (c computeInstancesClient) DeleteInstance(ctx context.Context, projectID, zone, name string) error {
	operation, err := c.service.Instances.Delete(projectID, zone, name).Context(ctx).Do()
	if err != nil {
		// An instance already gone is the state the caller wanted.
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			return nil
		}
		return err
	}

	// Compute Engine answers a delete with an operation, and the instance is
	// still there until it finishes. Returning early would let a later
	// resource type, such as a network, be deleted out from under it.
	return c.waitForZoneOperation(ctx, projectID, zone, operation)
}

// waitForZoneOperation polls one zone operation until it finishes.
func (c computeInstancesClient) waitForZoneOperation(ctx context.Context, projectID, zone string, operation *compute.Operation) error {
	if operation == nil {
		return nil
	}

	for {
		if operation.Status == "DONE" {
			if operation.Error != nil && len(operation.Error.Errors) > 0 {
				return fmt.Errorf("%s: %s", operation.Error.Errors[0].Code, operation.Error.Errors[0].Message)
			}
			return nil
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		answer, err := c.service.ZoneOperations.Wait(projectID, zone, operation.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("waiting for the delete to finish: %w", err)
		}
		operation = answer
	}
}

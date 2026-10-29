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
	runv1 "google.golang.org/api/run/v1"
	runv2 "google.golang.org/api/run/v2"
)

// CloudRunServicesAPI is the part of the Cloud Run Admin API this resource
// uses. The v2 list call refuses the "-" wildcard for a location, so a run has
// to ask each region in turn, which is a decision about what to ask and what to
// do when one region refuses. Those live in the lister, and the SDK clients and
// the long-running delete live in the adapter at the bottom of this file.
type CloudRunServicesAPI interface {
	// ListLocations returns the regions where this project can run Cloud Run.
	ListLocations(ctx context.Context, projectID string) ([]string, error)

	// ListServices returns the services in one region.
	ListServices(ctx context.Context, projectID, location string) ([]CloudRunService, error)

	// DeleteService deletes one service by its full name and waits for the
	// operation to finish. Deleting one that is already gone is not an error.
	DeleteService(ctx context.Context, name string) error
}

// CloudRunService is what this resource needs about one service.
type CloudRunService struct {
	// Name is the full resource name,
	// projects/{project}/locations/{location}/services/{service}, which is what
	// the delete takes.
	Name string

	// Location is the region the service runs in. It is also in Name, but the
	// lister reads it from here so a fake does not have to encode it twice.
	Location string

	Created time.Time
	Labels  map[string]string
}

// NewCloudRunServices creates the Cloud Run service resource.
func NewCloudRunServices() GcpResource {
	// gcpCfg is filled in by InitClient and read by the lister, which is how a
	// resource sees the location filter: the generic Lister signature carries
	// the resource's own config, not the provider's.
	gcpCfg := &GcpConfig{}

	// The lister asks one region at a time, so it reads the location filter
	// itself rather than letting the adapter drop the answers afterwards. That
	// is the difference between one call and forty on a run narrowed to one
	// region.
	return NewGcpResourceWithLocationScope(&resource.Resource[CloudRunServicesAPI]{
		ResourceTypeName: "cloud-run-service",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[CloudRunServicesAPI], cfg GcpConfig) {
			*gcpCfg = cfg
			r.Scope.ProjectID = cfg.ProjectID

			// Both clients are built once here. Building one per call, which an
			// earlier attempt at this resource did, spends a round trip on
			// every region a run visits.
			locations, err := runv1.NewService(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Cloud Run location client: %w", err)
				return
			}
			services, err := runv2.NewService(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Cloud Run client: %w", err)
				return
			}
			r.Client = cloudRunServicesClient{locations: locations, services: services}
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

// listCloudRunServices returns the services in the regions the location filter
// allows and the config filters keep.
func listCloudRunServices(ctx context.Context, client CloudRunServicesAPI, scope resource.Scope, cfg config.ResourceType, gcpCfg GcpConfig) ([]*string, error) {
	locations, err := client.ListLocations(ctx, scope.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("error listing Cloud Run locations: %w", err)
	}

	var asked []string
	for _, location := range locations {
		if gcpCfg.MatchesLocationFilter(location) {
			asked = append(asked, location)
		}
	}
	if len(asked) == 0 {
		return nil, nil
	}

	var result []*string
	var refused []string
	var lastErr error

	for _, location := range asked {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		services, err := client.ListServices(ctx, scope.ProjectID, location)
		if err != nil {
			// One region refusing must not cost the rest of the run, which is
			// the whole point of asking them one at a time.
			logging.Debugf("Could not list Cloud Run services in %s: %v", location, err)
			refused = append(refused, location)
			lastErr = err
			continue
		}

		for _, service := range services {
			name := service.Name
			created := service.Created

			// A config rule reads the short name, which is what the console
			// shows and what somebody writing a rule types, while the
			// identifier stays the full name the delete needs.
			short := shortResourceName(name)
			value := config.ResourceValue{Name: &short, Tags: service.Labels}
			if !created.IsZero() {
				value.Time = &created
			}

			if cfg.ShouldInclude(value) {
				result = append(result, &name)
			}
		}
	}

	// Every region refusing is not a project with no services. It is a run that
	// learned nothing, and answering zero would tell the engineer the project
	// is clean.
	if len(refused) == len(asked) {
		return nil, fmt.Errorf("could not list Cloud Run services in any of the %d regions asked for, the last failure in %s: %w", len(asked), refused[len(refused)-1], lastErr)
	}

	if len(refused) > 0 {
		logging.Warnf("Could not list Cloud Run services in %d of %d regions (%s), so any service there is left alone.", len(refused), len(asked), strings.Join(refused, ", "))
	}

	return result, nil
}

// shortResourceName returns the last segment of a full GCP resource name, which
// is the name an engineer reads in the console and writes in a config rule.
func shortResourceName(name string) string {
	if at := strings.LastIndex(name, "/"); at >= 0 {
		return name[at+1:]
	}
	return name
}

// deleteCloudRunService deletes one service by its full name.
func deleteCloudRunService(ctx context.Context, client CloudRunServicesAPI, name *string) error {
	if err := client.DeleteService(ctx, *name); err != nil {
		return fmt.Errorf("error deleting Cloud Run service %s: %w", *name, err)
	}

	logging.Debugf("Deleted Cloud Run service: %s", *name)
	return nil
}

// cloudRunServicesClient drives the SDK: the two API versions this resource
// needs, and the long-running delete.
type cloudRunServicesClient struct {
	// locations is the v1 client, because listing a project's Cloud Run
	// regions is a v1 call. The v2 API has no equivalent.
	locations *runv1.APIService

	// services is the v2 client, which is the current API for a service.
	services *runv2.Service
}

func (c cloudRunServicesClient) ListLocations(ctx context.Context, projectID string) ([]string, error) {
	var found []string

	call := c.locations.Projects.Locations.List("projects/" + projectID)
	if err := call.Pages(ctx, func(page *runv1.ListLocationsResponse) error {
		for _, location := range page.Locations {
			if location != nil && location.LocationId != "" {
				found = append(found, location.LocationId)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return found, nil
}

func (c cloudRunServicesClient) ListServices(ctx context.Context, projectID, location string) ([]CloudRunService, error) {
	var found []CloudRunService

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	call := c.services.Projects.Locations.Services.List(parent)
	if err := call.Pages(ctx, func(page *runv2.GoogleCloudRunV2ListServicesResponse) error {
		for _, service := range page.Services {
			if service == nil {
				continue
			}

			created, err := time.Parse(time.RFC3339, service.CreateTime)
			if err != nil {
				// A service whose age cannot be read is still a service; the
				// config layer refuses it under a time filter rather than this
				// dropping it here.
				logging.Debugf("Could not read the creation time of Cloud Run service %s: %v", service.Name, err)
				created = time.Time{}
			}

			found = append(found, CloudRunService{
				Name:     service.Name,
				Location: location,
				Created:  created,
				Labels:   service.Labels,
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return found, nil
}

func (c cloudRunServicesClient) DeleteService(ctx context.Context, name string) error {
	operation, err := c.services.Projects.Locations.Services.Delete(name).Context(ctx).Do()
	if err != nil {
		// A service already gone is the state the caller wanted.
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			return nil
		}
		return err
	}

	// Cloud Run answers a delete with an operation, and the service is still
	// there until it finishes. Returning early would report a deleted service
	// that a later list still answers with.
	return c.waitForOperation(ctx, operation)
}

// cloudRunOperationWait is how long each wait call blocks server side. The API
// answers as soon as the operation finishes, so this is the ceiling on one
// call rather than a delay the run always pays.
const cloudRunOperationWait = 30 * time.Second

// waitForOperation asks the API to hold the connection open until the operation
// finishes, and asks again while it has not. The call is explicitly best
// effort and may answer early, so one answer is not proof it is done.
func (c cloudRunServicesClient) waitForOperation(ctx context.Context, operation *runv2.GoogleLongrunningOperation) error {
	if operation == nil {
		return nil
	}

	for !operation.Done {
		if err := ctx.Err(); err != nil {
			return err
		}

		request := &runv2.GoogleLongrunningWaitOperationRequest{Timeout: fmt.Sprintf("%ds", int(cloudRunOperationWait.Seconds()))}
		answer, err := c.services.Projects.Locations.Operations.Wait(operation.Name, request).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("waiting for the delete to finish: %w", err)
		}
		operation = answer
	}

	if operation.Error != nil {
		return fmt.Errorf("%d: %s", operation.Error.Code, operation.Error.Message)
	}

	return nil
}

var _ CloudRunServicesAPI = cloudRunServicesClient{}

package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"google.golang.org/api/googleapi"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

// CloudSQLInstancesAPI is the part of the Cloud SQL Admin API this resource
// uses. The SDK answers a list with a pager and a delete with a long-running
// operation, neither of which a test can build, so both live in the adapter at
// the bottom of this file and the decisions stay testable.
type CloudSQLInstancesAPI interface {
	// ListInstances returns every Cloud SQL instance in the project.
	ListInstances(ctx context.Context, projectID string) ([]CloudSQLInstance, error)

	// DeleteInstance deletes one instance and waits for the operation to
	// finish. Deleting one that is already gone is not an error. The project
	// is the client's, since a Cloud SQL instance identifier is its bare name.
	DeleteInstance(ctx context.Context, name string) error
}

// CloudSQLInstance is what this resource needs about one database instance.
type CloudSQLInstance struct {
	Name string

	// Region is where the instance runs, such as us-central1. A Cloud SQL
	// instance name carries no location, so this is read off the API's answer
	// and this resource filters on it itself.
	Region string

	Created time.Time
	Labels  map[string]string

	// Protected is true when the instance carries deletion protection, which
	// makes the delete fail until somebody turns it off.
	Protected bool
}

// NewCloudSQLInstances creates the Cloud SQL instance resource.
func NewCloudSQLInstances() GcpResource {
	// gcpCfg is filled in by InitClient and read by the lister, which is how a
	// resource sees the location filter: the generic Lister signature carries
	// the resource's own config, not the provider's.
	gcpCfg := &GcpConfig{}

	// An instance's name carries no location, so its lister reads the region
	// off the API's answer and applies the filter itself.
	return NewGcpResourceWithLocationScope(&resource.Resource[CloudSQLInstancesAPI]{
		ResourceTypeName: "cloud-sql-instance",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[CloudSQLInstancesAPI], cfg GcpConfig) {
			*gcpCfg = cfg
			r.Scope.ProjectID = cfg.ProjectID

			service, err := sqladmin.NewService(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Cloud SQL client: %w", err)
				return
			}
			r.Client = cloudSQLInstancesClient{service: service, projectID: cfg.ProjectID}
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GcpCloudSQLInstance
		},
		Lister: func(ctx context.Context, client CloudSQLInstancesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return listCloudSQLInstances(ctx, client, scope, cfg, *gcpCfg)
		},
		Nuker: resource.SequentialDeleter(deleteCloudSQLInstance),
	}, LocationAppliedByLister)
}

// listCloudSQLInstances returns the instances in the project that match the
// config filters and the location filter.
func listCloudSQLInstances(ctx context.Context, client CloudSQLInstancesAPI, scope resource.Scope, cfg config.ResourceType, gcpCfg GcpConfig) ([]*string, error) {
	instances, err := client.ListInstances(ctx, scope.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("error listing Cloud SQL instances: %w", err)
	}

	protected := 0

	var result []*string
	for _, instance := range instances {
		if !gcpCfg.MatchesLocationFilter(instance.Region) {
			continue
		}

		// A protected instance refuses its delete until somebody turns the
		// protection off, so the run says it is there rather than reporting a
		// failure per instance during the nuke.
		if instance.Protected {
			logging.Debugf("Skipping Cloud SQL instance %s: it carries deletion protection", instance.Name)
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
			result = append(result, &name)
		}
	}

	if protected > 0 {
		logging.Warnf("Leaving %d Cloud SQL instances alone: they carry deletion protection, which Cloud SQL refuses to delete through.", protected)
	}

	return result, nil
}

// deleteCloudSQLInstance deletes one instance by name.
func deleteCloudSQLInstance(ctx context.Context, client CloudSQLInstancesAPI, name *string) error {
	if err := client.DeleteInstance(ctx, *name); err != nil {
		return fmt.Errorf("error deleting Cloud SQL instance %s: %w", *name, err)
	}

	logging.Debugf("Deleted Cloud SQL instance: %s", *name)
	return nil
}

// cloudSQLInstancesClient drives the SDK: the pager, the long-running delete
// and the fields Cloud SQL reports about an instance.
type cloudSQLInstancesClient struct {
	service *sqladmin.Service

	// projectID is held here because an instance's identifier is its bare
	// name, so the delete has nowhere else to read the project from.
	projectID string
}

func (c cloudSQLInstancesClient) ListInstances(ctx context.Context, projectID string) ([]CloudSQLInstance, error) {
	var found []CloudSQLInstance

	err := c.service.Instances.List(projectID).Pages(ctx, func(page *sqladmin.InstancesListResponse) error {
		for _, instance := range page.Items {
			if instance == nil {
				continue
			}

			created, err := time.Parse(time.RFC3339, instance.CreateTime)
			if err != nil {
				// An instance whose age cannot be read is still an instance;
				// the config layer refuses it under a time filter rather than
				// this dropping it here.
				logging.Debugf("Could not read the creation time of Cloud SQL instance %s: %v", instance.Name, err)
				created = time.Time{}
			}

			sqlInstance := CloudSQLInstance{
				Name:    instance.Name,
				Region:  instance.Region,
				Created: created,
			}
			if instance.Settings != nil {
				sqlInstance.Labels = instance.Settings.UserLabels
				sqlInstance.Protected = instance.Settings.DeletionProtectionEnabled
			}

			found = append(found, sqlInstance)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return found, nil
}

func (c cloudSQLInstancesClient) DeleteInstance(ctx context.Context, name string) error {
	operation, err := c.service.Instances.Delete(c.projectID, name).Context(ctx).Do()
	if err != nil {
		// An instance already gone is the state the caller wanted.
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			return nil
		}
		return err
	}

	// Cloud SQL answers a delete with an operation, and the instance is still
	// there until it finishes. Returning early would report a deleted instance
	// that a later list still answers with.
	return c.waitForOperation(ctx, operation)
}

// waitForOperation polls one operation until Cloud SQL reports it finished.
func (c cloudSQLInstancesClient) waitForOperation(ctx context.Context, operation *sqladmin.Operation) error {
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

		// The Admin API offers no wait call, so this asks again on a timer.
		if err := sleep(ctx, cloudSQLOperationPoll); err != nil {
			return err
		}

		answer, err := c.service.Operations.Get(c.projectID, operation.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("waiting for the delete to finish: %w", err)
		}
		operation = answer
	}
}

// cloudSQLOperationPoll is how long the adapter waits between asking whether a
// delete has finished. A Cloud SQL delete takes minutes, so asking faster
// spends quota to learn nothing.
const cloudSQLOperationPoll = 10 * time.Second

// sleep waits, and answers the cancellation instead when the run is stopped.
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

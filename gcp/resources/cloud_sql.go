package resources

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
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

	// MasterInstanceName is set on a read replica and names the instance it
	// replicates. Cloud SQL refuses to delete an instance that still has a
	// replica, so the lister returns replicas ahead of their primary.
	MasterInstanceName string
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

	var eligible []CloudSQLInstance
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
			eligible = append(eligible, instance)
		}
	}

	if protected > 0 {
		logging.Warnf("Leaving %d Cloud SQL instances alone: they carry deletion protection, which Cloud SQL refuses to delete through.", protected)
	}

	eligible = replicasBeforeTheirPrimary(eligible, instances)
	warnAboutReplicasLeftBehind(eligible, instances)

	result := make([]*string, 0, len(eligible))
	for _, instance := range eligible {
		name := instance.Name
		result = append(result, &name)
	}

	return result, nil
}

// replicasBeforeTheirPrimary orders the instances a run will delete so that no
// instance comes before a replica of it. Cloud SQL refuses to delete an
// instance that still has a replica, and the nuker works through identifiers one
// at a time, so the primary's delete fails outright in the other order. The
// chain is walked over every instance in the project, not just the eligible
// ones, since a replica a filter left out still sits between its primary and
// the top of the chain.
func replicasBeforeTheirPrimary(eligible, all []CloudSQLInstance) []CloudSQLInstance {
	masterOf := make(map[string]string, len(all))
	for _, instance := range all {
		if instance.MasterInstanceName != "" {
			masterOf[instance.Name] = instanceNameOnly(instance.MasterInstanceName)
		}
	}
	if len(masterOf) == 0 {
		return eligible
	}

	ordered := make([]CloudSQLInstance, len(eligible))
	copy(ordered, eligible)

	// A stable sort keeps the API's order among instances at the same depth,
	// which is what a run without replicas reports today.
	sort.SliceStable(ordered, func(i, j int) bool {
		return replicaDepth(ordered[i].Name, masterOf) > replicaDepth(ordered[j].Name, masterOf)
	})
	return ordered
}

// replicaDepth counts the replication hops between an instance and the primary
// at the top of its chain: a primary is 0, a replica of it 1, and a replica of
// that replica 2. Deleting the deepest first is the order Cloud SQL accepts.
func replicaDepth(name string, masterOf map[string]string) int {
	// Cloud SQL cannot build a replication cycle, but a malformed answer that
	// described one would otherwise spin here forever.
	seen := map[string]bool{name: true}

	depth := 0
	for {
		master := masterOf[name]
		if master == "" || seen[master] {
			return depth
		}
		seen[master] = true
		name = master
		depth++
	}
}

// warnAboutReplicasLeftBehind says so when a run will try to delete an instance
// whose replica is not in the same run. Cloud SQL refuses that delete, and the
// filter that left the replica out is the reason: a replica can sit in another
// region, which puts it outside a run narrowed by --region.
func warnAboutReplicasLeftBehind(eligible, all []CloudSQLInstance) {
	inRun := make(map[string]bool, len(eligible))
	for _, instance := range eligible {
		inRun[instance.Name] = true
	}

	for _, instance := range all {
		master := instanceNameOnly(instance.MasterInstanceName)
		if master == "" || inRun[instance.Name] || !inRun[master] {
			continue
		}
		logging.Warnf("Cloud SQL instance %s has a replica this run leaves alone (%s), and Cloud SQL refuses to delete an instance that still has one.", master, instance.Name)
	}
}

// instanceNameOnly drops the project qualifier Cloud SQL puts on the primary
// named by a replica, so that project:db and db name the same instance.
func instanceNameOnly(name string) string {
	if at := strings.LastIndex(name, ":"); at >= 0 {
		return name[at+1:]
	}
	return name
}

// deleteCloudSQLInstance deletes one instance by name.
func deleteCloudSQLInstance(ctx context.Context, client CloudSQLInstancesAPI, name *string) error {
	if err := client.DeleteInstance(ctx, *name); err != nil {
		return fmt.Errorf("error deleting Cloud SQL instance %s: %w", *name, err)
	}

	logging.Debugf("Deleted Cloud SQL instance: %s", *name)
	return nil
}

// incompleteList reads the warnings Cloud SQL puts on a successful list to say
// the answer is missing instances, and turns one into the error the run should
// have seen. A nuke that reported success here would leave those instances
// behind without saying so.
func incompleteList(warnings []*sqladmin.ApiWarning) error {
	for _, warning := range warnings {
		if warning == nil {
			continue
		}

		switch warning.Code {
		case "REGION_UNREACHABLE", "MAX_RESULTS_EXCEEDS_LIMIT":
			where := warning.Region
			if where == "" {
				where = "the project"
			}
			return fmt.Errorf("incomplete list of Cloud SQL instances, %s for %s: %s", warning.Code, where, warning.Message)
		}
	}

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
		// Cloud SQL reports an unreachable region as a warning on an otherwise
		// successful answer. Reading past one would let the run report a clean
		// sweep of a project whose missing region still holds databases.
		if err := incompleteList(page.Warnings); err != nil {
			return err
		}

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
				Name:               instance.Name,
				Region:             instance.Region,
				Created:            created,
				MasterInstanceName: instance.MasterInstanceName,
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

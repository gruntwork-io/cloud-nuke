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

// NewCloudRunJobs creates a new Cloud Run job resource using the generic resource pattern.
func NewCloudRunJobs() GcpResource {
	return NewGcpResource(&resource.Resource[*run.JobsClient]{
		ResourceTypeName: "cloud-run-job",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*run.JobsClient], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := run.NewJobsClient(context.Background())
			if err != nil {
				panic(fmt.Sprintf("failed to create Cloud Run jobs client: %v", err))
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GcpCloudRunJob
		},
		Lister: listCloudRunJobs,
		Nuker:  resource.SequentialDeleter(deleteCloudRunJob),
	})
}

// listCloudRunJobs retrieves all Cloud Run jobs across all regions in the project
// that match the config filters. Locations are enumerated via listCloudRunLocations and
// queried individually — the Cloud Run API does not support the "locations/-" wildcard.
func listCloudRunJobs(ctx context.Context, client *run.JobsClient, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	locations, err := listCloudRunLocations(ctx, scope.ProjectID)
	if err != nil {
		return nil, err
	}

	var result []*string

	for _, location := range locations {
		parent := fmt.Sprintf("projects/%s/locations/%s", scope.ProjectID, location)

		it := client.ListJobs(ctx, &runpb.ListJobsRequest{Parent: parent})
		for {
			job, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				// Some locations may not have Cloud Run enabled — skip them rather than
				// aborting the entire listing.
				logging.Debugf("error listing Cloud Run jobs in %s, skipping location: %v", location, err)
				break
			}

			// Extract the short name for config filtering; the full resource name is
			// retained as the delete identifier since the API requires it.
			shortName := job.Name[strings.LastIndex(job.Name, "/")+1:]

			var resourceTime time.Time
			if job.GetCreateTime() != nil {
				resourceTime = job.GetCreateTime().AsTime()
			}

			labels := job.GetLabels()
			if labels == nil {
				labels = map[string]string{}
			}

			resourceValue := config.ResourceValue{
				Name: &shortName,
				Time: &resourceTime,
				Tags: labels,
			}

			if cfg.ShouldInclude(resourceValue) {
				name := job.Name
				result = append(result, &name)
			}
		}
	}

	return result, nil
}

// deleteCloudRunJob deletes a single Cloud Run job and waits for the operation to complete.
// Any active executions are automatically cancelled by the API before the job is deleted.
func deleteCloudRunJob(ctx context.Context, client *run.JobsClient, name *string) error {
	jobName := *name

	op, err := client.DeleteJob(ctx, &runpb.DeleteJobRequest{Name: jobName})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Cloud Run job %s already deleted, skipping", jobName)
			return nil
		}
		return fmt.Errorf("error deleting Cloud Run job %s: %w", jobName, err)
	}

	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("error waiting for Cloud Run job %s deletion: %w", jobName, err)
	}

	logging.Debugf("Deleted Cloud Run job: %s", jobName)
	return nil
}

package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	functions "cloud.google.com/go/functions/apiv2"
	"cloud.google.com/go/functions/apiv2/functionspb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewCloudFunctions creates a new Cloud Functions resource using the generic resource pattern.
func NewCloudFunctions() GcpResource {
	return NewGcpResource(&resource.Resource[*functions.FunctionClient]{
		ResourceTypeName: "cloud-function",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*functions.FunctionClient], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := functions.NewFunctionClient(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Cloud Functions client: %w", err)
				return
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudFunction
		},
		Lister: listCloudFunctions,
		Nuker:  resource.SequentialDeleter(deleteCloudFunction),
	})
}

// listCloudFunctions retrieves all Cloud Functions (Gen2) in the project that match the config filters.
// It queries across all locations using the wildcard parent: projects/{projectID}/locations/-
func listCloudFunctions(ctx context.Context, client *functions.FunctionClient, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	// Use the wildcard "-" for location to list functions across ALL regions
	parent := fmt.Sprintf("projects/%s/locations/-", scope.ProjectID)

	req := &functionspb.ListFunctionsRequest{
		Parent: parent,
	}

	it := client.ListFunctions(ctx, req)
	for {
		fn, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing cloud functions: %w", err)
		}

		// fn.Name is the fully qualified name: projects/{project}/locations/{location}/functions/{function}
		name := fn.Name

		// Use UpdateTime as the resource timestamp for time-based filtering
		var resourceTime time.Time
		if fn.UpdateTime != nil {
			resourceTime = fn.UpdateTime.AsTime()
		}

		resourceValue := config.ResourceValue{
			Name: &name,
			Time: &resourceTime,
		}

		if cfg.ShouldInclude(resourceValue) {
			result = append(result, &name)
		}
	}

	return result, nil
}

// Rate limiting delay between function deletions to avoid API quota issues
const cloudFunctionDeleteDelay = 5 * time.Second

// deleteCloudFunction deletes a single Cloud Function.
func deleteCloudFunction(ctx context.Context, client *functions.FunctionClient, name *string) error {
	functionName := *name

	req := &functionspb.DeleteFunctionRequest{
		Name: functionName,
	}

	op, err := client.DeleteFunction(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Cloud function %s already deleted, skipping", functionName)
			return nil
		}
		return fmt.Errorf("error deleting cloud function %s: %w", functionName, err)
	}

	// Wait for the long-running delete operation to complete
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("error waiting for cloud function %s deletion: %w", functionName, err)
	}

	logging.Debugf("Deleted Cloud Function: %s", functionName)

	// Rate limiting delay to avoid API quota issues
	time.Sleep(cloudFunctionDeleteDelay)

	return nil
}

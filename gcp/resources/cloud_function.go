package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
			r.Scope.Locations = cfg.Locations
			r.Scope.ExcludeLocations = cfg.ExcludeLocations
			client, err := functions.NewFunctionClient(context.Background())
			if err != nil {
				// Panic is recovered by GcpResourceAdapter.Init() and stored as InitializationError,
				// causing subsequent GetAndSetIdentifiers/Nuke calls to return the error gracefully.
				panic(fmt.Sprintf("failed to create Cloud Functions client: %v", err))
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
// If scope.Locations is set, it queries specific locations; otherwise it uses the wildcard locations/-.
func listCloudFunctions(ctx context.Context, client *functions.FunctionClient, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	// Build the list of parents to query
	var parents []string
	if len(scope.Locations) == 0 {
		// Use the wildcard "-" for location to list functions across ALL locations
		parents = []string{fmt.Sprintf("projects/%s/locations/-", scope.ProjectID)}
	} else {
		for _, loc := range scope.Locations {
			if !MatchesLocationFilter(loc, nil, scope.ExcludeLocations) {
				continue
			}
			if strings.EqualFold(loc, "global") {
				logging.Debugf("Skipping Cloud Functions for location 'global' (not supported)")
				continue
			}
			parents = append(parents, fmt.Sprintf("projects/%s/locations/%s", scope.ProjectID, loc))
		}
	}

	for _, parent := range parents {
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
				logging.Debugf("Error listing cloud functions in %s: %v", parent, err)
				break
			}

			// fn.Name is the fully qualified name: projects/{project}/locations/{location}/functions/{function}
			name := fn.Name

			// Post-filter by location when using wildcard query
			if len(scope.ExcludeLocations) > 0 {
				loc := ExtractLocationFromResourceName(name)
				if loc == "" || !MatchesLocationFilter(loc, nil, scope.ExcludeLocations) {
					continue
				}
			}

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

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

// CloudFunctionsAPI is the part of the Cloud Functions API this resource uses.
// The SDK's client answers a list with an iterator whose internals are
// unexported, so a test cannot build one. Listing and deleting are therefore
// expressed in plain values here, and the SDK iteration lives in the one
// adapter below, which leaves the filtering and the names a test can reach.
type CloudFunctionsAPI interface {
	// ListFunctions returns every function under the parent, across locations.
	ListFunctions(ctx context.Context, parent string) ([]CloudFunction, error)

	// DeleteFunction deletes one function and waits for the operation to
	// finish. Deleting one that is already gone is not an error.
	DeleteFunction(ctx context.Context, name string) error
}

// CloudFunction is what this resource needs to know about one function: its
// fully qualified name, which carries the location, and when it last changed.
type CloudFunction struct {
	Name    string
	Updated time.Time
}

// NewCloudFunctions creates a new Cloud Functions resource using the generic resource pattern.
func NewCloudFunctions() GcpResource {
	return NewGcpResource(&resource.Resource[CloudFunctionsAPI]{
		ResourceTypeName: "cloud-function",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[CloudFunctionsAPI], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := functions.NewFunctionClient(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create Cloud Functions client: %w", err)
				return
			}
			r.Client = cloudFunctionsClient{client: client}
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudFunction
		},
		Lister: listCloudFunctions,
		Nuker:  resource.SequentialDeleter(deleteCloudFunction),
	})
}

// listCloudFunctions retrieves all Cloud Functions (Gen2) in the project that
// match the config filters, across every location.
func listCloudFunctions(ctx context.Context, client CloudFunctionsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// The wildcard "-" for location lists functions in every region at once.
	parent := fmt.Sprintf("projects/%s/locations/-", scope.ProjectID)

	functions, err := client.ListFunctions(ctx, parent)
	if err != nil {
		return nil, fmt.Errorf("error listing cloud functions: %w", err)
	}

	var result []*string
	for _, function := range functions {
		name := function.Name
		updated := function.Updated

		if cfg.ShouldInclude(config.ResourceValue{Name: &name, Time: &updated}) {
			result = append(result, &name)
		}
	}

	return result, nil
}

// deleteCloudFunction deletes a single Cloud Function.
func deleteCloudFunction(ctx context.Context, client CloudFunctionsAPI, name *string) error {
	if err := client.DeleteFunction(ctx, *name); err != nil {
		return fmt.Errorf("error deleting cloud function %s: %w", *name, err)
	}

	logging.Debugf("Deleted Cloud Function: %s", *name)
	return nil
}

// cloudFunctionsClient drives the SDK. It holds the parts a test cannot reach:
// the iterator, the long-running operation and the rate-limiting delay.
type cloudFunctionsClient struct {
	client *functions.FunctionClient
}

// cloudFunctionDeleteDelay keeps deletions inside the API's quota.
const cloudFunctionDeleteDelay = 5 * time.Second

func (c cloudFunctionsClient) ListFunctions(ctx context.Context, parent string) ([]CloudFunction, error) {
	var found []CloudFunction

	it := c.client.ListFunctions(ctx, &functionspb.ListFunctionsRequest{Parent: parent})
	for {
		fn, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return found, nil
		}
		if err != nil {
			return nil, err
		}

		// fn.Name is projects/{project}/locations/{location}/functions/{function},
		// and UpdateTime is the only timestamp the API reports.
		function := CloudFunction{Name: fn.Name}
		if fn.UpdateTime != nil {
			function.Updated = fn.UpdateTime.AsTime()
		}
		found = append(found, function)
	}
}

func (c cloudFunctionsClient) DeleteFunction(ctx context.Context, name string) error {
	op, err := c.client.DeleteFunction(ctx, &functionspb.DeleteFunctionRequest{Name: name})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			logging.Debugf("Cloud function %s already deleted, skipping", name)
			return nil
		}
		return err
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("waiting for deletion: %w", err)
	}

	// The function is gone by here, so the run must report it deleted. This
	// delay only keeps the next delete inside the API's quota, and a cancelled
	// run stops waiting rather than reporting a deleted function as failed.
	select {
	case <-time.After(cloudFunctionDeleteDelay):
	case <-ctx.Done():
	}

	return nil
}

var _ CloudFunctionsAPI = cloudFunctionsClient{}

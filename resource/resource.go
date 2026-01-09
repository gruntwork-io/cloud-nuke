package resource

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// DefaultBatchSize is the maximum number of resources per batch
const DefaultBatchSize = 10

// Scope represents the cloud provider-specific scope for a resource.
// For AWS: Region is set (e.g., "us-east-1" or "global" for global resources)
// For GCP: ProjectID is set, and optionally Region for regional resources
type Scope struct {
	Region    string // AWS region (e.g., "us-east-1") or "global" for global resources
	ProjectID string // GCP project ID
}

// String returns a human-readable representation of the scope for logging
func (s Scope) String() string {
	if s.ProjectID != "" {
		if s.Region != "" {
			return s.ProjectID + "/" + s.Region
		}
		return s.ProjectID
	}
	return s.Region
}

// Resource is the universal struct for all nukeable resources.
// C is the cloud service client type (e.g., *ec2.Client, *storage.Client).
//
// This single struct contains:
// - Configuration: what kind of resource this is and how to interact with it
// - Runtime state: client, scope, discovered identifiers, nukable status
//
// Usage: Create with struct literal, then call Init() before other methods.
type Resource[C any] struct {
	// === Configuration (set at construction time) ===

	// ResourceTypeName is the unique identifier (e.g., "ec2-keypairs", "gcs-bucket")
	ResourceTypeName string

	// BatchSize is the maximum number of resources to delete per batch.
	// If 0, defaults to DefaultBatchSize (10).
	// Set based on AWS/GCP API rate limits for this resource type.
	BatchSize int

	// IsGlobal indicates whether this resource is global (true) or regional (false).
	// Global resources (e.g., IAM, Route53) are only queried once, not per-region.
	// Regional resources (e.g., EC2, S3) are queried for each target region.
	IsGlobal bool

	// InitClient initializes the client from cloud-specific config.
	// For AWS: cfg is aws.Config
	// For GCP: cfg is string (projectID)
	// Set r.Client and r.Scope directly in this function.
	InitClient func(r *Resource[C], cfg any)

	// ConfigGetter retrieves the resource-specific config section
	ConfigGetter func(c config.Config) config.ResourceType

	// Lister retrieves all resource identifiers to nuke.
	// Receives the resource-specific config (extracted via ConfigGetter).
	Lister func(ctx context.Context, client C, scope Scope, resourceCfg config.ResourceType) ([]*string, error)

	// Nuker deletes the resources. Use SimpleBatchDeleter, SequentialDeleter, or MultiStepDeleter.
	Nuker NukerFunc[C]

	// PermissionVerifier performs optional dry-run permission checks (nil = skip verification)
	PermissionVerifier func(ctx context.Context, client C, id *string) error

	// === Runtime state (set during execution) ===

	// Client is the typed cloud service client
	Client C

	// Scope contains Region (AWS) and/or ProjectID (GCP)
	Scope Scope

	// identifiers holds the discovered resource IDs
	identifiers []string

	// nukables tracks which resources can be nuked (nil value = nukable)
	nukables map[string]error
}

// Init initializes the resource with cloud-specific configuration.
// For AWS: cfg should be aws.Config
// For GCP: cfg should be string (projectID)
// Must be called before GetAndSetIdentifiers or Nuke.
func (r *Resource[C]) Init(cfg any) {
	r.nukables = make(map[string]error)
	if r.InitClient != nil {
		r.InitClient(r, cfg)
	}
}

// ResourceName returns the unique resource type name (implements AwsResource/GcpResource interface)
func (r *Resource[C]) ResourceName() string {
	return r.ResourceTypeName
}

// ResourceIdentifiers returns the currently stored identifiers (implements AwsResource/GcpResource interface)
func (r *Resource[C]) ResourceIdentifiers() []string {
	return r.identifiers
}

// MaxBatchSize returns the batch size for this resource (implements AwsResource/GcpResource interface)
func (r *Resource[C]) MaxBatchSize() int {
	if r.BatchSize > 0 {
		return r.BatchSize
	}
	return DefaultBatchSize
}

// GetAndSetResourceConfig retrieves the resource-specific configuration (implements AwsResource/GcpResource interface)
func (r *Resource[C]) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	if r.ConfigGetter == nil {
		return config.ResourceType{}
	}
	return r.ConfigGetter(configObj)
}

// GetAndSetIdentifiers discovers resources and stores their identifiers (implements AwsResource/GcpResource interface)
func (r *Resource[C]) GetAndSetIdentifiers(ctx context.Context, configObj config.Config) ([]string, error) {
	if r.Lister == nil {
		return nil, fmt.Errorf("%s: Lister function not configured", r.ResourceTypeName)
	}

	// Extract resource-specific config and pass to Lister
	if r.ConfigGetter == nil {
		return nil, fmt.Errorf("%s: ConfigGetter function not configured", r.ResourceTypeName)
	}
	resourceCfg := r.ConfigGetter(configObj)
	identifiers, err := r.Lister(ctx, r.Client, r.Scope, resourceCfg)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to list resources in %s: %w", r.ResourceTypeName, r.Scope, err)
	}

	// Run permission verification if configured
	if r.PermissionVerifier != nil {
		r.verifyNukablePermissions(identifiers, func(id *string) error {
			return r.PermissionVerifier(ctx, r.Client, id)
		})
	}

	r.identifiers = aws.ToStringSlice(identifiers)
	return r.identifiers, nil
}

// Nuke deletes the resources with the given identifiers (implements AwsResource/GcpResource interface)
// Returns results for each identifier. Caller handles reporting.
func (r *Resource[C]) Nuke(ctx context.Context, identifiers []string) []NukeResult {
	if len(identifiers) == 0 {
		return nil
	}

	if r.Nuker == nil {
		// Return error result for all identifiers
		results := make([]NukeResult, len(identifiers))
		err := fmt.Errorf("%s: Nuker function not configured", r.ResourceTypeName)
		for i, id := range identifiers {
			results[i] = NukeResult{Identifier: id, Error: err}
		}
		return results
	}

	ptrIdentifiers := util.ToStringPtrSlice(identifiers)
	results := r.Nuker(ctx, r.Client, r.Scope, r.ResourceTypeName, ptrIdentifiers)

	// Log results
	for _, result := range results {
		if result.Error != nil {
			logging.Errorf("[Failed] %s %s: %s", r.ResourceTypeName, result.Identifier, result.Error)
		} else {
			logging.Debugf("[OK] Deleted %s: %s", r.ResourceTypeName, result.Identifier)
		}
	}

	return results
}

// IsNukable checks if a resource can be nuked (implements AwsResource/GcpResource interface).
// Returns (true, nil) if nukable, (false, error) if not.
// If the identifier was never verified, returns (true, nil) - assuming nukable by default.
func (r *Resource[C]) IsNukable(id string) (bool, error) {
	err, ok := r.nukables[id]
	if !ok {
		// Not in the map - for resources without permission verification,
		// this is the normal case. Return nukable.
		return true, nil
	}
	if err != nil {
		// Explicitly marked as not nukable
		return false, err
	}
	// Explicitly marked as nukable (verified with nil error)
	return true, nil
}

// setNukableStatus sets the nukable status for an identifier
func (r *Resource[C]) setNukableStatus(id string, err error) {
	if r.nukables == nil {
		r.nukables = make(map[string]error)
	}
	r.nukables[id] = err
}

// verifyNukablePermissions checks permissions for each ID using the provided check function
func (r *Resource[C]) verifyNukablePermissions(ids []*string, checkFn func(id *string) error) {
	if r.nukables == nil {
		r.nukables = make(map[string]error)
	}
	for _, id := range ids {
		if id == nil {
			continue
		}
		idStr := *id
		// Skip if already verified
		if _, exists := r.nukables[idStr]; exists {
			continue
		}
		err := checkFn(id)
		r.setNukableStatus(idStr, err)
	}
}

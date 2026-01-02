// Package resources provides AWS resource implementations for cloud-nuke.
//
// This package contains the adapter layer that bridges the generic resource.Resource[C]
// pattern with AWS-specific types. The adapter allows generic resources to satisfy
// the AwsResource interface required by cloud-nuke's AWS execution engine.
//
// # Architecture
//
// The generic Resource[C] type (in resource package) provides:
//   - Type-safe client handling via generics
//   - Standardized listing, filtering, and deletion patterns
//   - Centralized error handling and reporting
//
// The adapter layer (this package) provides:
//   - WrapAwsInitClient: Type-safe aws.Config initialization helper
//   - AwsResourceAdapter: Wraps Resource[C] to implement AwsResource interface
//   - NewAwsResource: Factory function to create adapted resources
//
// # Creating a New AWS Resource
//
// Define your resource using the generic pattern with WrapAwsInitClient:
//
//	func NewMyResource() AwsResource {
//	    return NewAwsResource(&resource.Resource[MyClientAPI]{
//	        ResourceTypeName: "my-resource",
//	        BatchSize:        50,
//	        InitClient: WrapAwsInitClient(func(r *resource.Resource[MyClientAPI], cfg aws.Config) {
//	            r.Scope.Region = cfg.Region
//	            r.Client = myservice.NewFromConfig(cfg)
//	        }),
//	        // ... other configuration
//	    })
//	}
package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

const (
	// DefaultWaitTimeout is the default timeout for AWS resource deletion waiters.
	DefaultWaitTimeout = 5 * time.Minute

	// maxBatchSize is the default maximum batch size for resource operations.
	// This is set to 49 as a safe maximum across most AWS API limits.
	maxBatchSize = 49

	// firstSeenTagKey is a tag used to track resource creation time for resources
	// that don't have a native created-at timestamp (e.g., EIP, ECS Clusters).
	// This supports the `--older-than <duration>` filtering in cloud-nuke.
	firstSeenTagKey = "cloud-nuke-first-seen"
)

// AwsInitClientFunc is the AWS-specific client initialization function signature.
// It receives a typed aws.Config instead of any, providing type safety and
// eliminating boilerplate type assertions in each resource implementation.
//
// The function should:
//   - Set r.Client to the appropriate AWS service client
//   - Set r.Scope.Region from cfg.Region (for regional resources)
//
// Example:
//
//	func(r *resource.Resource[*ec2.Client], cfg aws.Config) {
//	    r.Scope.Region = cfg.Region
//	    r.Client = ec2.NewFromConfig(cfg)
//	}
type AwsInitClientFunc[C any] func(r *resource.Resource[C], cfg aws.Config)

// WrapAwsInitClient converts an AwsInitClientFunc to the generic InitClient function
// signature required by resource.Resource. This centralizes the type assertion from
// any to aws.Config, keeping individual resource implementations clean.
//
// This is the recommended way to set InitClient for all AWS resources:
//
//	InitClient: WrapAwsInitClient(func(r *resource.Resource[MyAPI], cfg aws.Config) {
//	    r.Scope.Region = cfg.Region
//	    r.Client = myservice.NewFromConfig(cfg)
//	}),
//
// The wrapper panics on type assertion failures since this indicates a programming
// error - the AWS execution engine should always pass aws.Config. Panicking ensures
// such errors are caught immediately during development and testing.
func WrapAwsInitClient[C any](fn AwsInitClientFunc[C]) func(r *resource.Resource[C], cfg any) {
	return func(r *resource.Resource[C], cfg any) {
		awsCfg, ok := cfg.(aws.Config)
		if !ok {
			panic(fmt.Sprintf("WrapAwsInitClient: expected aws.Config, got %T (this is a bug in cloud-nuke)", cfg))
		}
		fn(r, awsCfg)
	}
}

// AwsResourceAdapter wraps a generic resource.Resource[C] to satisfy the AwsResource
// interface required by cloud-nuke's AWS execution engine.
//
// The adapter delegates most methods directly to the embedded Resource, with Init()
// being the key adaptation point where aws.Config is passed to the generic Init(any).
//
// This adapter should not be constructed directly. Use NewAwsResource() instead.
type AwsResourceAdapter[C any] struct {
	*resource.Resource[C]
}

// NewAwsResource creates an AwsResourceAdapter from a generic Resource.
// This is the factory function that should be used to create AWS resources
// that follow the generic resource pattern.
//
// Example:
//
//	func NewEC2Instances() AwsResource {
//	    return NewAwsResource(&resource.Resource[EC2InstancesAPI]{
//	        ResourceTypeName: "ec2",
//	        InitClient: WrapAwsInitClient(...),
//	        // ... other fields
//	    })
//	}
func NewAwsResource[C any](r *resource.Resource[C]) AwsResource {
	return &AwsResourceAdapter[C]{Resource: r}
}

// Init initializes the resource with AWS configuration.
// This method bridges the AwsResource.Init(aws.Config) interface to the generic
// Resource.Init(any) method, allowing the AWS execution engine to initialize
// resources with typed configuration.
func (a *AwsResourceAdapter[C]) Init(cfg aws.Config) {
	a.Resource.Init(cfg)
}

// Nuke deletes the resources with the given identifiers.
// Delegates directly to the embedded Resource's Nuke method.
func (a *AwsResourceAdapter[C]) Nuke(ctx context.Context, identifiers []string) error {
	return a.Resource.Nuke(ctx, identifiers)
}

// PrepareContext is a no-op for generic resources.
// The generic Resource pattern passes context directly to Nuke() and Lister functions,
// eliminating the need for context preparation. This method exists solely for
// compatibility with the AwsResource interface.
func (a *AwsResourceAdapter[C]) PrepareContext(_ context.Context, _ config.ResourceType) error {
	return nil
}

// Compile-time interface satisfaction check.
// This ensures AwsResourceAdapter correctly implements AwsResource at compile time.
var _ AwsResource = (*AwsResourceAdapter[any])(nil)

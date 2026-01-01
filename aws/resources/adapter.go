package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AwsInitClientFunc is the AWS-specific client initialization function.
// This receives the typed aws.Config instead of any, eliminating boilerplate
// type assertion in each resource.
type AwsInitClientFunc[C any] func(r *resource.Resource[C], cfg aws.Config)

// WrapAwsInitClient converts an AwsInitClientFunc to the generic InitClient function.
// This handles the type assertion from any to aws.Config in one place.
func WrapAwsInitClient[C any](fn AwsInitClientFunc[C]) func(r *resource.Resource[C], cfg any) {
	return func(r *resource.Resource[C], cfg any) {
		awsCfg, ok := cfg.(aws.Config)
		if !ok {
			logging.Debugf("Invalid config type for AWS client: expected aws.Config, got %T", cfg)
			return
		}
		fn(r, awsCfg)
	}
}

// AwsResourceAdapter wraps a generic Resource to satisfy the AwsResource interface.
// It provides type-safe Init(aws.Config) for AWS configuration initialization.
type AwsResourceAdapter[C any] struct {
	*resource.Resource[C]
}

// NewAwsResource creates an AwsResourceAdapter from a generic Resource.
func NewAwsResource[C any](r *resource.Resource[C]) AwsResource {
	return &AwsResourceAdapter[C]{Resource: r}
}

// Init initializes the resource with AWS config.
// Sets the region in scope from the config.
func (a *AwsResourceAdapter[C]) Init(cfg aws.Config) {
	a.Resource.Init(cfg)
}

// Nuke deletes the resources with the given identifiers.
func (a *AwsResourceAdapter[C]) Nuke(ctx context.Context, identifiers []string) error {
	return a.Resource.Nuke(ctx, identifiers)
}

// PrepareContext is a no-op for generic resources since context is passed directly to Nuke.
// This exists for compatibility with the AwsResource interface.
func (a *AwsResourceAdapter[C]) PrepareContext(_ context.Context, _ config.ResourceType) error {
	return nil
}

// Compile-time interface satisfaction check
var _ AwsResource = (*AwsResourceAdapter[any])(nil)

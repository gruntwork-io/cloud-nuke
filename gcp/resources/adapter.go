package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/gruntwork-io/cloud-nuke/resource"
)

const (
	DefaultWaitTimeout = 5 * time.Minute
	DefaultBatchSize   = 50
)

// GcpInitClientFunc is the type-safe client initialization function signature.
type GcpInitClientFunc[C any] func(r *resource.Resource[C], projectID string)

// WrapGcpInitClient converts a GcpInitClientFunc to the generic InitClient signature.
// Panics on type assertion failure since that indicates a programming error.
func WrapGcpInitClient[C any](fn GcpInitClientFunc[C]) func(r *resource.Resource[C], cfg any) {
	return func(r *resource.Resource[C], cfg any) {
		projectID, ok := cfg.(string)
		if !ok {
			panic(fmt.Sprintf("WrapGcpInitClient: expected string projectID, got %T", cfg))
		}
		fn(r, projectID)
	}
}

// GcpResourceAdapter wraps a generic Resource to satisfy the GcpResource interface.
type GcpResourceAdapter[C any] struct {
	*resource.Resource[C]
}

// NewGcpResource creates a GcpResourceAdapter from a generic Resource.
func NewGcpResource[C any](r *resource.Resource[C]) GcpResource {
	return &GcpResourceAdapter[C]{Resource: r}
}

// Init initializes the resource with GCP project ID.
func (g *GcpResourceAdapter[C]) Init(projectID string) {
	g.Resource.Init(projectID)
}

// Nuke deletes the resources with the given identifiers.
func (g *GcpResourceAdapter[C]) Nuke(ctx context.Context, identifiers []string) error {
	return g.Resource.Nuke(ctx, identifiers)
}

var _ GcpResource = (*GcpResourceAdapter[any])(nil)

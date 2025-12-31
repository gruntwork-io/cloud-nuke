package resources

import (
	"context"

	"github.com/gruntwork-io/cloud-nuke/resource"
)

// GcpResourceAdapter wraps a generic Resource to satisfy the GcpResource interface.
// It provides type-safe Init(string) for GCP's project ID initialization.
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

// Compile-time interface satisfaction check
var _ GcpResource = (*GcpResourceAdapter[any])(nil)

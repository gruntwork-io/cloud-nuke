package resources

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
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
	// initErr stores any error (including recovered panics) from Init so that
	// subsequent operations can fail gracefully instead of crashing.
	initErr error
}

// NewGcpResource creates a GcpResourceAdapter from a generic Resource.
func NewGcpResource[C any](r *resource.Resource[C]) GcpResource {
	return &GcpResourceAdapter[C]{Resource: r}
}

// Init initializes the resource with GCP project ID.
// Recovers from panics in InitClient (e.g., credential failures) and stores
// the error so GetAndSetIdentifiers and Nuke fail gracefully.
func (g *GcpResourceAdapter[C]) Init(projectID string) {
	defer func() {
		if r := recover(); r != nil {
			logging.Debugf("Recovered panic during Init of %s: %v\n%s",
				g.ResourceTypeName, r, debug.Stack())
			g.initErr = fmt.Errorf("initialization failed: %v", r)
		}
	}()
	g.Resource.Init(projectID)
}

// GetAndSetIdentifiers discovers resources and stores their identifiers.
// Returns an error if Init failed.
func (g *GcpResourceAdapter[C]) GetAndSetIdentifiers(ctx context.Context, configObj config.Config) ([]string, error) {
	if g.initErr != nil {
		return nil, fmt.Errorf("%s: %w", g.ResourceName(), g.initErr)
	}
	return g.Resource.GetAndSetIdentifiers(ctx, configObj)
}

// Nuke deletes the resources with the given identifiers.
// Returns an error if Init failed.
func (g *GcpResourceAdapter[C]) Nuke(ctx context.Context, identifiers []string) ([]resource.NukeResult, error) {
	if g.initErr != nil {
		return nil, fmt.Errorf("%s: %w", g.ResourceName(), g.initErr)
	}
	return g.Resource.Nuke(ctx, identifiers)
}

var _ GcpResource = (*GcpResourceAdapter[any])(nil)

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

const (
	DefaultWaitTimeout = 5 * time.Minute
	DefaultBatchSize   = 50
)

// GcpConfig holds the configuration needed to initialize a GCP resource,
// mirroring how AWS uses aws.Config as the init argument.
type GcpConfig struct {
	ProjectID string
	Region    string
}

// GcpInitClientFunc is the type-safe client initialization function signature.
type GcpInitClientFunc[C any] func(r *resource.Resource[C], cfg GcpConfig)

// WrapGcpInitClient converts a GcpInitClientFunc to the generic InitClient signature.
// Panics on type assertion failure since that indicates a programming error.
func WrapGcpInitClient[C any](fn GcpInitClientFunc[C]) func(r *resource.Resource[C], cfg any) {
	return func(r *resource.Resource[C], cfg any) {
		gcpCfg, ok := cfg.(GcpConfig)
		if !ok {
			panic(fmt.Sprintf("WrapGcpInitClient: expected GcpConfig, got %T", cfg))
		}
		fn(r, gcpCfg)
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

// Init initializes the resource with GCP configuration.
// Panics from InitClient (e.g., missing credentials) are recovered and stored
// in Resource.InitializationError so that subsequent GetAndSetIdentifiers/Nuke
// calls return the error gracefully instead of crashing the process.
func (g *GcpResourceAdapter[C]) Init(cfg GcpConfig) {
	defer func() {
		if r := recover(); r != nil {
			g.Resource.InitializationError = fmt.Errorf("panic during Init for %s: %v", g.Resource.ResourceTypeName, r)
			logging.Debugf("%s", g.Resource.InitializationError)
		}
	}()
	g.Resource.Init(cfg)
}

// Nuke deletes the resources with the given identifiers.
func (g *GcpResourceAdapter[C]) Nuke(ctx context.Context, identifiers []string) ([]resource.NukeResult, error) {
	return g.Resource.Nuke(ctx, identifiers)
}

var _ GcpResource = (*GcpResourceAdapter[any])(nil)

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	DefaultWaitTimeout = 5 * time.Minute
	DefaultBatchSize   = 50
)

// GcpConfig holds the configuration needed to initialize a GCP resource.
type GcpConfig struct {
	ProjectID        string
	Locations        []string
	ExcludeLocations []string
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
			g.InitializationError = errors.WithStackTrace(fmt.Errorf("panic during Init for %s: %v", g.ResourceTypeName, r))
			logging.Debugf("%s", g.InitializationError)
		}
	}()
	g.Resource.Init(cfg)
}

// Nuke deletes the resources with the given identifiers.
func (g *GcpResourceAdapter[C]) Nuke(ctx context.Context, identifiers []string) ([]resource.NukeResult, error) {
	return g.Resource.Nuke(ctx, identifiers)
}

// MatchesLocationFilter checks whether a given location matches the location
// filter expressed by the Locations and ExcludeLocations in GcpConfig.
// If no locations are specified, everything matches.
func MatchesLocationFilter(location string, locations []string, excludeLocations []string) bool {
	// Check exclusions first
	for _, exc := range excludeLocations {
		if strings.EqualFold(exc, location) {
			return false
		}
	}

	// If no include filter, everything matches
	if len(locations) == 0 {
		return true
	}

	// Check if this location is in the include list
	for _, loc := range locations {
		if strings.EqualFold(loc, location) {
			return true
		}
	}

	return false
}

// ExtractLocationFromResourceName extracts the location from a GCP fully qualified resource name
// that contains a /locations/{location}/ segment. Returns empty string if not found.
// Works for any resource: projects/{p}/locations/{loc}/functions/{f}, .../secrets/{s}, etc.
func ExtractLocationFromResourceName(name string) string {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "locations" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

var _ GcpResource = (*GcpResourceAdapter[any])(nil)

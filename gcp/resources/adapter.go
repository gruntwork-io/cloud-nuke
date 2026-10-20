package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

const (
	DefaultWaitTimeout = 5 * time.Minute
	DefaultBatchSize   = 50
)

// LocationScope says how a resource type's location is decided, which is what
// lets the adapter apply the --region filter in one place instead of trusting
// every lister to remember it. Every registered resource declares one, and
// TestEveryResourceDeclaresALocationScope fails when a new one does not.
type LocationScope int

const (
	// LocationFromResourceName is the default: the identifier is a full GCP
	// resource name carrying /locations/<location>/, so the adapter reads the
	// location out of it and filters.
	LocationFromResourceName LocationScope = iota

	// LocationAppliedByLister means the lister has already applied the filter,
	// because the location is on the API's response rather than in the
	// identifier. A GCS bucket is the case: its name carries no location.
	LocationAppliedByLister

	// LocationNone means the resource type has no location at all, such as a
	// Pub/Sub topic. Asking for one location cannot include it, so the adapter
	// skips the whole type when a location filter is set, and says so.
	LocationNone
)

// GcpConfig holds the configuration needed to initialize a GCP resource,
// mirroring how AWS uses aws.Config as the init argument.
type GcpConfig struct {
	ProjectID string

	// Locations and ExcludeLocations are the --region filter. A GCP project is
	// scanned in one pass, so a location narrows what a lister keeps rather
	// than deciding which scope the engine walks.
	Locations        []string
	ExcludeLocations []string
}

// HasLocationFilter reports whether the user narrowed the run to locations.
func (c GcpConfig) HasLocationFilter() bool {
	return len(c.Locations) > 0 || len(c.ExcludeLocations) > 0
}

// MatchesLocationFilter reports whether a resource in the given location
// survives the filter the user asked for. No filter keeps everything, and an
// exclusion beats an inclusion.
func (c GcpConfig) MatchesLocationFilter(location string) bool {
	for _, excluded := range c.ExcludeLocations {
		if strings.EqualFold(excluded, location) {
			return false
		}
	}

	if len(c.Locations) == 0 {
		return true
	}

	for _, wanted := range c.Locations {
		if strings.EqualFold(wanted, location) {
			return true
		}
	}

	return false
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

// GcpResourceAdapter wraps a generic Resource to satisfy the GcpResource
// interface, and owns the --region filter for every GCP resource.
type GcpResourceAdapter[C any] struct {
	*resource.Resource[C]

	locationScope LocationScope

	// cfg is what Init was given. The adapter reads it when it filters, which
	// is why the resource itself never has to capture it.
	cfg GcpConfig
}

// NewGcpResource creates a GcpResourceAdapter whose identifiers carry their own
// location, which is true of every resource whose identifier is a full GCP
// resource name.
func NewGcpResource[C any](r *resource.Resource[C]) GcpResource {
	return NewGcpResourceWithLocationScope(r, LocationFromResourceName)
}

// NewGcpResourceWithLocationScope creates a GcpResourceAdapter for a resource
// whose location is decided some other way.
func NewGcpResourceWithLocationScope[C any](r *resource.Resource[C], scope LocationScope) GcpResource {
	adapter := &GcpResourceAdapter[C]{Resource: r, locationScope: scope}

	// The adapter wraps the lister so the location filter runs in one audited
	// place. A lister that forgets the filter therefore cannot widen a run.
	if scope == LocationFromResourceName {
		lister := r.Lister
		r.Lister = func(ctx context.Context, client C, s resource.Scope, cfg config.ResourceType) ([]*string, error) {
			identifiers, err := lister(ctx, client, s, cfg)
			if err != nil {
				return nil, err
			}
			return adapter.keepMatchingLocations(identifiers)
		}
	}

	return adapter
}

// LocationScope reports how this resource type's location is decided.
func (g *GcpResourceAdapter[C]) LocationScope() LocationScope {
	return g.locationScope
}

// keepMatchingLocations drops the identifiers whose location the user filtered
// out. A resource of this kind is declared as carrying its location in its
// name, so an identifier without one is a bug in its lister: under an active
// filter that fails the listing rather than guessing, because guessing either
// deletes something the user excluded or keeps something they asked for.
func (g *GcpResourceAdapter[C]) keepMatchingLocations(identifiers []*string) ([]*string, error) {
	if !g.cfg.HasLocationFilter() {
		return identifiers, nil
	}

	kept := make([]*string, 0, len(identifiers))
	for _, identifier := range identifiers {
		if identifier == nil {
			continue
		}

		location := LocationFromName(*identifier)
		if location == "" {
			return nil, fmt.Errorf(
				"%s is declared as carrying its location in its name, but %q has none, so --region cannot be applied to it",
				g.ResourceTypeName, *identifier)
		}

		if g.cfg.MatchesLocationFilter(location) {
			kept = append(kept, identifier)
		}
	}

	return kept, nil
}

// LocationFromName reads the location out of a full GCP resource name such as
// projects/p/locations/us-central1/functions/f, and returns an empty string
// when the name carries none.
func LocationFromName(name string) string {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "locations" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// Init initializes the resource with GCP configuration. A panic from client
// construction, which is what a missing credential looks like, is recovered and
// kept as the resource's initialization error, so one unreachable API reports
// itself instead of ending the run.
func (g *GcpResourceAdapter[C]) Init(cfg GcpConfig) {
	defer func() {
		if recovered := recover(); recovered != nil {
			g.InitializationError = fmt.Errorf("%s: %v", g.ResourceTypeName, recovered)
			logging.Debugf("Could not initialize %s: %v", g.ResourceTypeName, recovered)
		}
	}()

	g.cfg = cfg

	// A resource type with no location cannot be narrowed to one, so it says so
	// rather than ignoring --region and deleting project-wide. An exclusion is
	// different: "everywhere except europe-west1" still includes a resource
	// that is in no location, so the run keeps deleting it.
	if g.locationScope == LocationNone && len(cfg.Locations) > 0 {
		g.InitializationError = fmt.Errorf(
			"%s has no location, so --region cannot narrow it; drop --region or pass --exclude-resource-type %s",
			g.ResourceTypeName, g.ResourceTypeName)
		return
	}

	g.Resource.Init(cfg)
}

// Nuke deletes the resources with the given identifiers.
func (g *GcpResourceAdapter[C]) Nuke(ctx context.Context, identifiers []string) ([]resource.NukeResult, error) {
	return g.Resource.Nuke(ctx, identifiers)
}

var _ GcpResource = (*GcpResourceAdapter[any])(nil)

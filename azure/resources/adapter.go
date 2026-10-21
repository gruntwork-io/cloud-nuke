package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// DefaultBatchSize is how many Azure resources are deleted per batch.
const DefaultBatchSize = 50

// LocationScope says how a resource type's location is decided, which is what
// lets the adapter apply the --region filter in one place instead of trusting
// every lister to remember it.
type LocationScope int

const (
	// LocationReportedByLister is the default: the lister reads the location
	// off the API's answer, which is how ARM reports it, and hands it to the
	// adapter with each identifier.
	LocationReportedByLister LocationScope = iota

	// LocationNone means the resource type has no location, such as a
	// subscription-level role assignment. Asking for one location cannot
	// include it, so it refuses --region rather than deleting everywhere.
	LocationNone
)

// AzureConfig holds what an Azure resource needs to build its clients.
type AzureConfig struct {
	SubscriptionID string

	// Locations and ExcludeLocations are the --region filter. ARM list calls
	// are subscription-scoped and report each resource's location, so a
	// location narrows what a lister keeps rather than what it visits.
	Locations        []string
	ExcludeLocations []string
}

// HasLocationFilter reports whether the user narrowed the run to locations.
func (c AzureConfig) HasLocationFilter() bool {
	return len(c.Locations) > 0 || len(c.ExcludeLocations) > 0
}

// MatchesLocationFilter reports whether a resource in the given location
// survives the filter the user asked for. No filter keeps everything, and an
// exclusion beats an inclusion. Azure reports a location as "westeurope" while
// a user may type "West Europe", so both are folded to the same form.
func (c AzureConfig) MatchesLocationFilter(location string) bool {
	normalized := normalizeLocation(location)

	for _, excluded := range c.ExcludeLocations {
		if normalizeLocation(excluded) == normalized {
			return false
		}
	}

	if len(c.Locations) == 0 {
		return true
	}

	for _, wanted := range c.Locations {
		if normalizeLocation(wanted) == normalized {
			return true
		}
	}

	return false
}

// normalizeLocation lowercases a location and drops its spaces, so "West
// Europe", "westeurope" and "West europe" are one location.
func normalizeLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(location, " ", ""))
}

// AzureInitClientFunc is the Azure-specific client initialization signature.
type AzureInitClientFunc[C any] func(r *resource.Resource[C], cfg AzureConfig)

// WrapAzureInitClient converts an AzureInitClientFunc to the generic signature
// resource.Resource takes, which centralizes the type assertion.
func WrapAzureInitClient[C any](fn AzureInitClientFunc[C]) func(r *resource.Resource[C], cfg any) {
	return func(r *resource.Resource[C], cfg any) {
		azureCfg, ok := cfg.(AzureConfig)
		if !ok {
			panic(fmt.Sprintf("WrapAzureInitClient: expected AzureConfig, got %T (this is a bug in cloud-nuke)", cfg))
		}
		fn(r, azureCfg)
	}
}

// AzureResourceAdapter wraps a generic Resource to satisfy AzureResource, and
// owns the --region filter for every Azure resource.
type AzureResourceAdapter[C any] struct {
	*resource.Resource[C]

	locationScope LocationScope
	cfg           AzureConfig
}

// NewAzureResource creates an adapter for a resource whose lister reports each
// identifier's location.
func NewAzureResource[C any](r *resource.Resource[C]) AzureResource {
	return NewAzureResourceWithLocationScope(r, LocationReportedByLister)
}

// NewAzureResourceWithLocationScope creates an adapter for a resource whose
// location is decided some other way.
func NewAzureResourceWithLocationScope[C any](r *resource.Resource[C], scope LocationScope) AzureResource {
	return &AzureResourceAdapter[C]{Resource: r, locationScope: scope}
}

// LocationScope reports how this resource type's location is decided.
func (a *AzureResourceAdapter[C]) LocationScope() LocationScope {
	return a.locationScope
}

// Config returns what Init was given, which is how a lister reaches the
// location filter: the generic Lister signature carries the resource's own
// config, not the provider's.
func (a *AzureResourceAdapter[C]) Config() AzureConfig {
	return a.cfg
}

// Init initializes the resource with Azure configuration. A panic from client
// construction, which is what a missing credential looks like, is recovered and
// kept as the resource's initialization error, so one unreachable API reports
// itself instead of ending the run.
func (a *AzureResourceAdapter[C]) Init(cfg AzureConfig) {
	defer func() {
		if recovered := recover(); recovered != nil {
			a.InitializationError = fmt.Errorf("%s: %v", a.ResourceTypeName, recovered)
			logging.Debugf("Could not initialize %s: %v", a.ResourceTypeName, recovered)
		}
	}()

	a.cfg = cfg

	// A resource type with no location cannot be narrowed to one, so it says so
	// rather than ignoring --region and deleting across the subscription. An
	// exclusion is different: "everywhere except westeurope" still includes a
	// resource that is in no location.
	if a.locationScope == LocationNone && len(cfg.Locations) > 0 {
		a.InitializationError = fmt.Errorf(
			"%s has no location, so --region cannot narrow it; drop --region or pass --exclude-resource-type %s",
			a.ResourceTypeName, a.ResourceTypeName)
		return
	}

	a.Resource.Init(cfg)
}

// Nuke deletes the resources with the given identifiers.
func (a *AzureResourceAdapter[C]) Nuke(ctx context.Context, identifiers []string) ([]resource.NukeResult, error) {
	return a.Resource.Nuke(ctx, identifiers)
}

var _ AzureResource = (*AzureResourceAdapter[any])(nil)

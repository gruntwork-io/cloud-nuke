package resources

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	cloudinv "github.com/james00012/cloud-inventory"
	"github.com/james00012/cloud-inventory/generic"
)

// Generated is one resource type backed by a descriptor the inventory
// generator read out of Google's Discovery document, rather than by a file
// written for it. Everything cloud-nuke decides stays here: which resources to
// act on, and what an unreadable part of a project means.
type Generated struct {
	// TypeName is what this type goes by on the command line.
	TypeName string

	// Descriptor is what the generator produced. Nothing in it is hand written.
	Descriptor generic.Descriptor

	// BaseURL and NewHTTPClient reach the API. A test points them at a stand-in.
	BaseURL       string
	NewHTTPClient func(ctx context.Context) (*http.Client, error)

	// ConfigGetter is this type's section of the config file.
	ConfigGetter func(config.Config) config.ResourceType

	// ScopeOf turns a run's project and locations into the scope this type's
	// list needs, which the descriptor already describes the shape of.
	ScopeOf func(GcpConfig) generic.Scope

	// LocationScope is how this type's location is decided, as for any other.
	LocationScope LocationScope
}

// NewGeneratedResource wires a generated descriptor into cloud-nuke. Adding a
// resource type this way is a descriptor and a few lines naming its config
// section, instead of a file per type.
func NewGeneratedResource(g Generated) GcpResource {
	gcpCfg := &GcpConfig{}

	return NewGcpResourceWithLocationScope(&resource.Resource[*generic.Client]{
		ResourceTypeName: g.TypeName,
		BatchSize:        DefaultBatchSize,

		InitClient: WrapGcpInitClient(func(r *resource.Resource[*generic.Client], cfg GcpConfig) {
			*gcpCfg = cfg
			r.Scope.ProjectID = cfg.ProjectID

			httpClient, err := g.NewHTTPClient(context.Background())
			if err != nil {
				r.InitializationError = fmt.Errorf("failed to create a client for %s: %w", g.TypeName, err)
				return
			}
			r.Client = &generic.Client{BaseURL: g.BaseURL, HTTP: httpClient, Descriptor: g.Descriptor}
		}),

		ConfigGetter: g.ConfigGetter,

		Lister: func(ctx context.Context, client *generic.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return listGenerated(ctx, client, g, *gcpCfg, cfg)
		},

		Nuker: resource.SequentialDeleter(func(ctx context.Context, client *generic.Client, id *string) error {
			// The identity goes back exactly as the listing gave it: its
			// format belongs to the provider and nothing here reads it.
			return client.Delete(ctx, cloudinv.ID(*id))
		}),
	}, g.LocationScope)
}

// listGenerated reads the project and keeps what the run asked for.
//
// A location the library could not read is reported and the run carries on
// with the rest, which is cloud-nuke's call rather than the library's: not
// nuking the locations it can read would leave more behind than it saves. A
// test suite takes the same report and fails.
func listGenerated(ctx context.Context, client *generic.Client, g Generated, gcpCfg GcpConfig, cfg config.ResourceType) ([]*string, error) {
	// An error here is a scope the provider cannot address or a cancelled run,
	// never a location that refused. Carrying on would nuke against a listing
	// the operator stopped.
	found, incomplete, err := client.List(ctx, g.ScopeOf(gcpCfg))
	if err != nil {
		return nil, err
	}
	if incomplete.Any() {
		logging.Warnf("could not read %s in %s, so anything there is left alone: %s",
			g.TypeName, incomplete.String(), incomplete.Cause())
	}

	var keep []*string
	for _, one := range found {
		name := one.DisplayName()
		created := one.CreatedAt()

		value := config.ResourceValue{Name: &name, Tags: one.Tags()}
		// A type whose API reports no creation time has none to filter on, and
		// a nil time is how the config says so.
		if !created.IsZero() {
			value.Time = &created
		}
		if !cfg.ShouldInclude(value) {
			continue
		}

		id := string(one.ResourceID())
		keep = append(keep, &id)
	}
	return keep, nil
}

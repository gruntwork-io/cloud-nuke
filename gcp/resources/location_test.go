package resources

import (
	"context"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newLocationScopedResource builds a resource whose lister answers with full
// GCP resource names in two locations and ignores the location filter, which is
// what every lister did before the adapter owned the filter.
func newLocationScopedResource(listed ...string) GcpResource {
	return NewGcpResource(&resource.Resource[any]{
		ResourceTypeName: "fake-located",
		InitClient:       WrapGcpInitClient(func(r *resource.Resource[any], cfg GcpConfig) {}),
		ConfigGetter:     func(c config.Config) config.ResourceType { return config.ResourceType{} },
		Lister: func(ctx context.Context, client any, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			out := make([]*string, 0, len(listed))
			for _, name := range listed {
				out = append(out, &name)
			}
			return out, nil
		},
	})
}

func TestTheAdapterFiltersLocationsALlisterIgnored(t *testing.T) {
	t.Parallel()

	central := "projects/p/locations/us-central1/functions/one"
	east := "projects/p/locations/us-east1/functions/two"

	for _, tc := range []struct {
		name     string
		cfg      GcpConfig
		expected []string
	}{
		{
			name:     "no filter keeps everything",
			cfg:      GcpConfig{ProjectID: "p"},
			expected: []string{central, east},
		},
		{
			name:     "an included location keeps only its own",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}},
			expected: []string{central},
		},
		{
			name:     "an excluded location drops its own",
			cfg:      GcpConfig{ProjectID: "p", ExcludeLocations: []string{"us-central1"}},
			expected: []string{east},
		},
		{
			name:     "the filter ignores case, which GCP APIs do not agree on",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"US-CENTRAL1"}},
			expected: []string{central},
		},
		{
			name:     "an exclusion beats an inclusion",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}, ExcludeLocations: []string{"us-central1"}},
			expected: []string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newLocationScopedResource(central, east)
			r.Init(tc.cfg)

			identifiers, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, identifiers)
		})
	}
}

// TestAnIdentifierWithNoLocationFailsTheListingUnderAFilter pins that the
// adapter refuses to guess. A resource declared as carrying its location in its
// name, whose lister answers with a name that has none, is a bug in that
// lister, and guessing would either delete what the user excluded or keep what
// they asked for.
func TestAnIdentifierWithNoLocationFailsTheListingUnderAFilter(t *testing.T) {
	t.Parallel()

	r := newLocationScopedResource("projects/p/locations/us-central1/functions/one", "projects/p/topics/no-location")
	r.Init(GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}})

	_, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"projects/p/topics/no-location" has none`)

	// With no filter there is nothing to apply, so the same lister is fine.
	clean := newLocationScopedResource("projects/p/locations/us-central1/functions/one", "projects/p/topics/no-location")
	clean.Init(GcpConfig{ProjectID: "p"})
	identifiers, err := clean.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.NoError(t, err)
	assert.Len(t, identifiers, 2)
}

func TestALocationlessResourceRefusesALocationFilter(t *testing.T) {
	t.Parallel()

	build := func() GcpResource {
		return NewGcpResourceWithLocationScope(&resource.Resource[any]{
			ResourceTypeName: "fake-locationless",
			InitClient:       WrapGcpInitClient(func(r *resource.Resource[any], cfg GcpConfig) {}),
			ConfigGetter:     func(c config.Config) config.ResourceType { return config.ResourceType{} },
			Lister: func(ctx context.Context, client any, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
				name := "projects/p/topics/everywhere"
				return []*string{&name}, nil
			},
		}, LocationNone)
	}

	// With no filter it lists as usual.
	unfiltered := build()
	unfiltered.Init(GcpConfig{ProjectID: "p"})
	identifiers, err := unfiltered.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.NoError(t, err)
	assert.Equal(t, []string{"projects/p/topics/everywhere"}, identifiers)

	// With an include filter it refuses, rather than listing the whole project
	// and letting the user believe the filter applied.
	filtered := build()
	filtered.Init(GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}})
	_, err = filtered.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no location")

	// The refusal is typed, so the provider can leave the type out with a
	// warning instead of ending the run on it.
	var noLocation LocationFilterNotApplicableError
	require.ErrorAs(t, err, &noLocation)
	assert.Equal(t, "fake-locationless", noLocation.ResourceTypeName)

	// An exclusion is not a narrowing: "everywhere except us-central1" still
	// includes a resource that is in no location, so it keeps deleting.
	excluded := build()
	excluded.Init(GcpConfig{ProjectID: "p", ExcludeLocations: []string{"us-central1"}})
	identifiers, err = excluded.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.NoError(t, err)
	assert.Equal(t, []string{"projects/p/topics/everywhere"}, identifiers)
}

func TestLocationFromName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "us-central1", LocationFromName("projects/p/locations/us-central1/functions/f"))
	assert.Equal(t, "europe-west4", LocationFromName("projects/p/locations/europe-west4/repositories/r"))
	assert.Equal(t, "", LocationFromName("projects/p/topics/t"))
	assert.Equal(t, "", LocationFromName("my-bucket"))
	assert.Equal(t, "", LocationFromName("projects/p/locations"))
}

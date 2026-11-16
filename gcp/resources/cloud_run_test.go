package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCloudRun answers with the regions and services a test names, records
// which regions were asked, and can refuse a region the way the API does when
// Cloud Run is not reachable there.
type fakeCloudRun struct {
	locations    []string
	locationsErr error

	// services is keyed by region.
	services map[string][]CloudRunService

	// refuse is keyed by region, and holds the error that region answers.
	refuse map[string]error

	asked   []string
	deleted []string
}

func (f *fakeCloudRun) ListLocations(ctx context.Context, projectID string) ([]string, error) {
	if f.locationsErr != nil {
		return nil, f.locationsErr
	}
	return f.locations, nil
}

func (f *fakeCloudRun) ListServices(ctx context.Context, projectID, location string) ([]CloudRunService, error) {
	f.asked = append(f.asked, location)
	if err, refused := f.refuse[location]; refused {
		return nil, err
	}
	return f.services[location], nil
}

func (f *fakeCloudRun) DeleteService(ctx context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func serviceIn(location, name string) CloudRunService {
	return CloudRunService{
		Name:     fmt.Sprintf("projects/p/locations/%s/services/%s", location, name),
		Location: location,
	}
}

func TestCloudRunServices_ResourceName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "cloud-run-service", NewCloudRunServices().ResourceName())
}

// The v2 list call refuses the "-" wildcard, so a run asks each region in turn.
// A run narrowed to one region must ask that one region and no other: asking
// all of them and dropping the answers afterwards would spend a round trip per
// region Cloud Run offers.
func TestListCloudRunServicesAsksOnlyTheRegionsTheFilterAllows(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		cfg   GcpConfig
		asked []string
		kept  []string
	}{
		{
			name:  "no filter asks every region",
			cfg:   GcpConfig{ProjectID: "p"},
			asked: []string{"us-central1", "europe-west4"},
			kept: []string{
				"projects/p/locations/us-central1/services/one",
				"projects/p/locations/europe-west4/services/two",
			},
		},
		{
			name:  "a region asks only its own",
			cfg:   GcpConfig{ProjectID: "p", Locations: []string{"europe-west4"}},
			asked: []string{"europe-west4"},
			kept:  []string{"projects/p/locations/europe-west4/services/two"},
		},
		{
			name:  "an exclusion skips its own",
			cfg:   GcpConfig{ProjectID: "p", ExcludeLocations: []string{"europe-west4"}},
			asked: []string{"us-central1"},
			kept:  []string{"projects/p/locations/us-central1/services/one"},
		},
		{
			name:  "a region is matched however it is typed",
			cfg:   GcpConfig{ProjectID: "p", Locations: []string{"EUROPE-WEST4"}},
			asked: []string{"europe-west4"},
			kept:  []string{"projects/p/locations/europe-west4/services/two"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &fakeCloudRun{
				locations: []string{"us-central1", "europe-west4"},
				services: map[string][]CloudRunService{
					"us-central1":  {serviceIn("us-central1", "one")},
					"europe-west4": {serviceIn("europe-west4", "two")},
				},
			}

			identifiers, err := listCloudRunServices(context.Background(), client,
				resource.Scope{ProjectID: "p"}, config.ResourceType{}, tc.cfg)
			require.NoError(t, err)

			kept := make([]string, 0, len(identifiers))
			for _, id := range identifiers {
				kept = append(kept, *id)
			}
			assert.ElementsMatch(t, tc.kept, kept)
			assert.ElementsMatch(t, tc.asked, client.asked)
		})
	}
}

// A region the filter rules out is never asked, and a run whose filter rules
// out every region asks nothing rather than asking all of them.
func TestListCloudRunServicesAsksNothingWhenTheFilterRulesOutEveryRegion(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{locations: []string{"us-central1"}}

	identifiers, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{},
		GcpConfig{ProjectID: "p", Locations: []string{"asia-northeast1"}})
	require.NoError(t, err)

	assert.Empty(t, identifiers)
	assert.Empty(t, client.asked)
}

// One region refusing must not cost the rest of the run, which is the reason
// for asking them one at a time in the first place.
func TestListCloudRunServicesKeepsGoingWhenOneRegionRefuses(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{
		locations: []string{"us-central1", "europe-west4"},
		services: map[string][]CloudRunService{
			"europe-west4": {serviceIn("europe-west4", "two")},
		},
		refuse: map[string]error{"us-central1": errors.New("permission denied")},
	}

	identifiers, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)

	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/europe-west4/services/two", *identifiers[0])
}

// Every region refusing is not a project with no services. It is a run that
// learned nothing, and answering zero would report the project clean.
func TestListCloudRunServicesFailsWhenEveryRegionRefuses(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{
		locations: []string{"us-central1", "europe-west4"},
		refuse: map[string]error{
			"us-central1":  errors.New("permission denied"),
			"europe-west4": errors.New("permission denied"),
		},
	}

	_, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "any of the 2 regions")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestListCloudRunServicesReportsALocationFailure(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{locationsErr: errors.New("run.googleapis.com is not enabled")}

	_, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not enabled")
}

// A config rule is written against the name the console shows, not the full
// resource path, while the identifier has to stay the full name the delete
// takes.
func TestListCloudRunServicesFiltersOnTheShortNameAndKeepsTheFullOne(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{
		locations: []string{"us-central1"},
		services: map[string][]CloudRunService{
			"us-central1": {serviceIn("us-central1", "keep-me"), serviceIn("us-central1", "drop-me")},
		},
	}

	byName := config.ResourceType{IncludeRule: config.FilterRule{
		NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("^keep-")}},
	}}
	identifiers, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byName, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)

	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/us-central1/services/keep-me", *identifiers[0])
}

func TestListCloudRunServicesFiltersOnLabelsAndTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	labelled := serviceIn("us-central1", "keep-me")
	labelled.Labels = map[string]string{"env": "test"}
	labelled.Created = now
	plain := serviceIn("us-central1", "drop-me")
	plain.Created = now

	client := &fakeCloudRun{
		locations: []string{"us-central1"},
		services:  map[string][]CloudRunService{"us-central1": {labelled, plain}},
	}

	byLabel := config.ResourceType{IncludeRule: config.FilterRule{
		Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("^test$")}},
	}}
	identifiers, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byLabel, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/us-central1/services/keep-me", *identifiers[0])

	// A time rule has to keep the service it does not name. Asserting only
	// that the recent one goes would pass just as well if the lister never
	// handed the creation time over at all, since the config layer excludes a
	// resource whose time it does not know.
	old := serviceIn("us-central1", "old-one")
	old.Created = now.Add(-48 * time.Hour)
	aged := &fakeCloudRun{
		locations: []string{"us-central1"},
		services:  map[string][]CloudRunService{"us-central1": {labelled, old}},
	}

	cutoff := now.Add(-time.Hour)
	byTime := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: &cutoff}}
	identifiers, err = listCloudRunServices(context.Background(), aged,
		resource.Scope{ProjectID: "p"}, byTime, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/us-central1/services/old-one", *identifiers[0])
}

// A cancelled run stops asking regions rather than working through the rest of
// the list it had planned.
func TestListCloudRunServicesStopsOnCancellation(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{locations: []string{"us-central1", "europe-west4"}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := listCloudRunServices(ctx, client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, client.asked)
}

func TestNukeCloudRunServiceDeletesByFullName(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{}
	name := "projects/p/locations/us-central1/services/one"

	require.NoError(t, deleteCloudRunService(context.Background(), client, &name))
	assert.Equal(t, []string{name}, client.deleted)
}

func TestShortResourceNameReadsTheLastSegment(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "one", shortResourceName("projects/p/locations/us-central1/services/one"))
	assert.Equal(t, "bare", shortResourceName("bare"))
	assert.Equal(t, "", shortResourceName(""))
}

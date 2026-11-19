package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeArtifactRegistry answers with the locations and repositories a test
// names, can fail any of the locations, and can run a hook as each location is
// asked, which is how a test cancels a run part way through.
type fakeArtifactRegistry struct {
	locations   []string
	byLocation  map[string][]ArtifactRepository
	failIn      map[string]bool
	onList      func(parent string)
	parentsSeen []string
	deleted     []string
}

func (f *fakeArtifactRegistry) ListLocations(ctx context.Context, project string) ([]string, error) {
	return f.locations, nil
}

func (f *fakeArtifactRegistry) ListRepositories(ctx context.Context, parent string) ([]ArtifactRepository, error) {
	f.parentsSeen = append(f.parentsSeen, parent)
	if f.onList != nil {
		f.onList(parent)
	}
	if f.failIn[parent] {
		// A paged listing hands back the pages that arrived along with the
		// error that ended it, so the fake answers with both.
		return f.byLocation[parent], errors.New("the location is unavailable")
	}
	return f.byLocation[parent], nil
}

func (f *fakeArtifactRegistry) DeleteRepository(ctx context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func TestArtifactRegistry_ResourceName(t *testing.T) {
	assert.Equal(t, "artifact-registry", NewArtifactRegistryRepositories().ResourceName())
}

func TestArtifactRegistry_MaxBatchSize(t *testing.T) {
	assert.Equal(t, 50, NewArtifactRegistryRepositories().MaxBatchSize())
}

func TestListArtifactRegistryWalksEveryLocationItWasGiven(t *testing.T) {
	t.Parallel()

	client := &fakeArtifactRegistry{
		locations: []string{"us-central1", "europe-west4"},
		byLocation: map[string][]ArtifactRepository{
			"projects/p/locations/us-central1":  {{Name: "projects/p/locations/us-central1/repositories/one"}},
			"projects/p/locations/europe-west4": {{Name: "projects/p/locations/europe-west4/repositories/two"}},
		},
	}

	identifiers, err := listArtifactRegistryRepositories(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})
	require.NoError(t, err)

	// The API rejects a location wildcard, so each location is asked for in turn.
	assert.Equal(t, []string{
		"projects/p/locations/us-central1",
		"projects/p/locations/europe-west4",
	}, client.parentsSeen)
	require.Len(t, identifiers, 2)
}

func TestListArtifactRegistryKeepsGoingWhenOneLocationFails(t *testing.T) {
	t.Parallel()

	client := &fakeArtifactRegistry{
		locations: []string{"us-central1", "europe-west4"},
		failIn:    map[string]bool{"projects/p/locations/us-central1": true},
		byLocation: map[string][]ArtifactRepository{
			"projects/p/locations/europe-west4": {{Name: "projects/p/locations/europe-west4/repositories/two"}},
		},
	}

	identifiers, err := listArtifactRegistryRepositories(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})

	// One location answering an error must not cost the repositories in the others.
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/europe-west4/repositories/two", *identifiers[0])
}

func TestListArtifactRegistryLeavesTheGcrMigrationRepositoriesAlone(t *testing.T) {
	t.Parallel()

	client := &fakeArtifactRegistry{
		locations: []string{"us"},
		byLocation: map[string][]ArtifactRepository{
			"projects/p/locations/us": {
				{Name: "projects/p/locations/us/repositories/gcr.io"},
				{Name: "projects/p/locations/us/repositories/us.gcr.io"},
				{Name: "projects/p/locations/us/repositories/mine"},
			},
		},
	}

	identifiers, err := listArtifactRegistryRepositories(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})
	require.NoError(t, err)

	// Deleting a repository GCP created for the Container Registry migration
	// breaks gcr.io image pulls for the whole project.
	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/us/repositories/mine", *identifiers[0])
}

func TestListArtifactRegistryFiltersOnTheShortNameAndReturnsTheFullOne(t *testing.T) {
	t.Parallel()

	created := time.Now()
	client := &fakeArtifactRegistry{
		locations: []string{"us-central1"},
		byLocation: map[string][]ArtifactRepository{
			"projects/p/locations/us-central1": {
				{Name: "projects/p/locations/us-central1/repositories/keep", Created: created, Labels: map[string]string{"env": "test"}},
				{Name: "projects/p/locations/us-central1/repositories/drop", Created: created},
			},
		},
	}

	// Anchored, so the rule can match the short name and nothing else. A
	// lister that handed the filters the full name would exclude neither.
	cfg := config.ResourceType{ExcludeRule: config.FilterRule{
		NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("^drop$")}},
	}}
	identifiers, err := listArtifactRegistryRepositories(context.Background(), client,
		resource.Scope{ProjectID: "p"}, cfg)
	require.NoError(t, err)

	// The one that stays comes back as the full name the delete call needs,
	// not the short name the rule matched on.
	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/us-central1/repositories/keep", *identifiers[0])
}

func TestListArtifactRegistryKeepsWhatALocationReadBeforeItFailed(t *testing.T) {
	t.Parallel()

	client := &fakeArtifactRegistry{
		locations: []string{"us-central1", "europe-west4"},
		failIn:    map[string]bool{"projects/p/locations/us-central1": true},
		byLocation: map[string][]ArtifactRepository{
			"projects/p/locations/us-central1":  {{Name: "projects/p/locations/us-central1/repositories/read"}},
			"projects/p/locations/europe-west4": {{Name: "projects/p/locations/europe-west4/repositories/two"}},
		},
	}

	identifiers, err := listArtifactRegistryRepositories(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})

	// The listing is paged, so an error part way through still leaves the
	// pages that arrived. Discarding them would leave repositories behind
	// that the project had already named.
	require.NoError(t, err)
	require.Len(t, identifiers, 2)
	assert.Equal(t, "projects/p/locations/us-central1/repositories/read", *identifiers[0])
	assert.Equal(t, "projects/p/locations/europe-west4/repositories/two", *identifiers[1])
}

func TestListArtifactRegistryFailsWhenEveryLocationFails(t *testing.T) {
	t.Parallel()

	client := &fakeArtifactRegistry{
		locations: []string{"us-central1", "europe-west4"},
		failIn: map[string]bool{
			"projects/p/locations/us-central1":  true,
			"projects/p/locations/europe-west4": true,
		},
		byLocation: map[string][]ArtifactRepository{
			"projects/p/locations/us-central1": {{Name: "projects/p/locations/us-central1/repositories/read"}},
		},
	}

	identifiers, err := listArtifactRegistryRepositories(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})

	// A scan that finished in no location learned too little to report on,
	// and answering without an error would tell the engineer the project is
	// clean, or cleaner than it is.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "any of the 2 locations")
	assert.Nil(t, identifiers)
}

func TestListArtifactRegistryStopsWhenTheRunIsCancelledBetweenLocations(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &fakeArtifactRegistry{
		locations: []string{"us-central1", "europe-west4"},
		byLocation: map[string][]ArtifactRepository{
			"projects/p/locations/us-central1":  {{Name: "projects/p/locations/us-central1/repositories/one"}},
			"projects/p/locations/europe-west4": {{Name: "projects/p/locations/europe-west4/repositories/two"}},
		},
		// The engineer presses Ctrl+C while the first location answers.
		onList: func(string) { cancel() },
	}

	identifiers, err := listArtifactRegistryRepositories(ctx, client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})

	// The run says it was cancelled rather than handing back the first
	// location's repositories as if they were the whole project's.
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, identifiers)
	assert.Equal(t, []string{"projects/p/locations/us-central1"}, client.parentsSeen)
}

func TestListArtifactRegistryStopsWhenTheRunIsCancelledDuringTheLastLocation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &fakeArtifactRegistry{
		locations: []string{"us-central1", "europe-west4"},
		failIn:    map[string]bool{"projects/p/locations/europe-west4": true},
		byLocation: map[string][]ArtifactRepository{
			"projects/p/locations/us-central1": {{Name: "projects/p/locations/us-central1/repositories/one"}},
		},
		// The cancel lands while the last location answers, so its listing
		// ends in an error and no later pass is left to notice the cancel.
		onList: func(parent string) {
			if parent == "projects/p/locations/europe-west4" {
				cancel()
			}
		},
	}

	identifiers, err := listArtifactRegistryRepositories(ctx, client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{})

	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, identifiers)
}

func TestNukeArtifactRegistryDeletesByFullName(t *testing.T) {
	t.Parallel()

	client := &fakeArtifactRegistry{}
	name := "projects/p/locations/us-central1/repositories/one"
	require.NoError(t, deleteArtifactRegistryRepository(context.Background(), client, &name))
	assert.Equal(t, []string{name}, client.deleted)
}

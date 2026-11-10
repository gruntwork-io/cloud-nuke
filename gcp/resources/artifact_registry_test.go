package resources

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeArtifactRegistry answers with the locations and repositories a test
// names, and can fail one location to show the scan carries on.
type fakeArtifactRegistry struct {
	locations      []string
	byLocation     map[string][]ArtifactRepository
	failInLocation string
	parentsSeen    []string
	deleted        []string
}

func (f *fakeArtifactRegistry) ListLocations(ctx context.Context, project string) ([]string, error) {
	return f.locations, nil
}

func (f *fakeArtifactRegistry) ListRepositories(ctx context.Context, parent string) ([]ArtifactRepository, error) {
	f.parentsSeen = append(f.parentsSeen, parent)
	if f.failInLocation != "" && parent == f.failInLocation {
		return nil, errors.New("the location is unavailable")
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
		locations:      []string{"us-central1", "europe-west4"},
		failInLocation: "projects/p/locations/us-central1",
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

	cfg := config.ResourceType{ExcludeRule: config.FilterRule{
		Tags: map[string]config.Expression{},
	}}
	identifiers, err := listArtifactRegistryRepositories(context.Background(), client,
		resource.Scope{ProjectID: "p"}, cfg)
	require.NoError(t, err)

	// Both survive an empty tag rule, and each identifier is the full name the
	// delete call needs rather than the short name the filters matched.
	require.Len(t, identifiers, 2)
	assert.Equal(t, "projects/p/locations/us-central1/repositories/keep", *identifiers[0])
}

func TestNukeArtifactRegistryDeletesByFullName(t *testing.T) {
	t.Parallel()

	client := &fakeArtifactRegistry{}
	name := "projects/p/locations/us-central1/repositories/one"
	require.NoError(t, deleteArtifactRegistryRepository(context.Background(), client, &name))
	assert.Equal(t, []string{name}, client.deleted)
}

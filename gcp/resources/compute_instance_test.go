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

// fakeComputeInstances answers with the instances a test names and records what
// it was asked to delete.
type fakeComputeInstances struct {
	instances []ComputeInstance
	listErr   error

	deleted []string
}

func (f *fakeComputeInstances) ListInstances(ctx context.Context, projectID string) ([]ComputeInstance, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.instances, nil
}

func (f *fakeComputeInstances) DeleteInstance(ctx context.Context, projectID, zone, name string) error {
	f.deleted = append(f.deleted, projectID+"/"+zone+"/"+name)
	return nil
}

func TestComputeInstances_ResourceName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "compute-instance", NewComputeInstances().ResourceName())
}

// An engineer who writes --region us-central1 means every zone in it, and one
// who names a zone means that zone. Both are matched, and an exclusion of
// either beats an inclusion of the other, since an exclusion is what somebody
// wrote to keep a machine alive.
func TestListComputeInstancesMatchesAZoneAgainstTheRegionItIsIn(t *testing.T) {
	t.Parallel()

	client := &fakeComputeInstances{instances: []ComputeInstance{
		{Name: "in-central-a", Zone: "us-central1-a"},
		{Name: "in-central-b", Zone: "us-central1-b"},
		{Name: "in-west4", Zone: "europe-west4-a"},
	}}

	for _, tc := range []struct {
		name     string
		cfg      GcpConfig
		expected []string
	}{
		{
			name:     "no filter keeps every instance",
			cfg:      GcpConfig{ProjectID: "p"},
			expected: []string{"in-central-a", "in-central-b", "in-west4"},
		},
		{
			name:     "a region keeps every zone in it",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}},
			expected: []string{"in-central-a", "in-central-b"},
		},
		{
			name:     "a zone keeps that zone alone",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"us-central1-b"}},
			expected: []string{"in-central-b"},
		},
		{
			name:     "a region excluded takes its zones with it",
			cfg:      GcpConfig{ProjectID: "p", ExcludeLocations: []string{"us-central1"}},
			expected: []string{"in-west4"},
		},
		{
			name:     "a zone excluded leaves its neighbors",
			cfg:      GcpConfig{ProjectID: "p", ExcludeLocations: []string{"us-central1-a"}},
			expected: []string{"in-central-b", "in-west4"},
		},
		{
			name:     "an exclusion beats an inclusion of the region it sits in",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}, ExcludeLocations: []string{"us-central1-a"}},
			expected: []string{"in-central-b"},
		},
		{
			name:     "a location is matched however it is typed",
			cfg:      GcpConfig{ProjectID: "p", Locations: []string{"US-CENTRAL1"}},
			expected: []string{"in-central-a", "in-central-b"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identifiers, err := listComputeInstances(context.Background(), client,
				resource.Scope{ProjectID: "p"}, config.ResourceType{}, tc.cfg)
			require.NoError(t, err)

			kept := make([]string, 0, len(identifiers))
			for _, id := range identifiers {
				_, _, name, err := parseInstanceIdentifier(*id)
				require.NoError(t, err)
				kept = append(kept, name)
			}
			assert.ElementsMatch(t, tc.expected, kept)
		})
	}
}

// A protected instance refuses its delete until somebody turns the protection
// off, so the scan leaves it alone rather than reporting a failure per instance
// during the nuke.
func TestListComputeInstancesLeavesAProtectedInstanceAlone(t *testing.T) {
	t.Parallel()

	client := &fakeComputeInstances{instances: []ComputeInstance{
		{Name: "protected", Zone: "us-central1-a", Protected: true},
		{Name: "ordinary", Zone: "us-central1-a"},
	}}

	identifiers, err := listComputeInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)

	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/zones/us-central1-a/instances/ordinary", *identifiers[0])
}

func TestListComputeInstancesFiltersOnNameLabelsAndTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	client := &fakeComputeInstances{instances: []ComputeInstance{
		{Name: "keep-me", Zone: "us-central1-a", Created: now, Labels: map[string]string{"env": "test"}},
		{Name: "drop-me", Zone: "us-central1-a", Created: now},
	}}

	byName := config.ResourceType{IncludeRule: config.FilterRule{
		NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("^keep-")}},
	}}
	identifiers, err := listComputeInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byName, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Contains(t, *identifiers[0], "keep-me")

	// A label rule reads the instance's own labels, which the lister has to
	// hand the config layer or a run narrowed by label deletes nothing.
	byLabel := config.ResourceType{IncludeRule: config.FilterRule{
		Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("^test$")}},
	}}
	identifiers, err = listComputeInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byLabel, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Contains(t, *identifiers[0], "keep-me")

	// An instance older than the cutoff is excluded by it.
	cutoff := now.Add(-time.Hour)
	byTime := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: &cutoff}}
	identifiers, err = listComputeInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byTime, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	assert.Empty(t, identifiers)
}

func TestListComputeInstancesReportsAListFailure(t *testing.T) {
	t.Parallel()

	client := &fakeComputeInstances{listErr: errors.New("permission denied")}

	_, err := listComputeInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

// The identifier carries the zone because an instance name is unique only
// inside one, and the delete reads it back out rather than guessing.
func TestTheInstanceIdentifierCarriesWhatTheDeleteNeeds(t *testing.T) {
	t.Parallel()

	identifier := instanceIdentifier("my-project", "us-central1-a", "web-1")
	assert.Equal(t, "projects/my-project/zones/us-central1-a/instances/web-1", identifier)

	projectID, zone, name, err := parseInstanceIdentifier(identifier)
	require.NoError(t, err)
	assert.Equal(t, []string{"my-project", "us-central1-a", "web-1"}, []string{projectID, zone, name})

	for _, wrong := range []string{
		"web-1",
		"projects/p/locations/us-central1/instances/web-1",
		"projects/p/zones/us-central1-a/instances",
	} {
		_, _, _, err := parseInstanceIdentifier(wrong)
		assert.Error(t, err, wrong)
	}
}

func TestNukeComputeInstanceDeletesTheOneTheIdentifierNames(t *testing.T) {
	t.Parallel()

	client := &fakeComputeInstances{}
	identifier := instanceIdentifier("my-project", "us-central1-a", "web-1")

	require.NoError(t, deleteComputeInstance(context.Background(), client, &identifier))
	assert.Equal(t, []string{"my-project/us-central1-a/web-1"}, client.deleted)

	// An identifier of another shape is a defect in the lister, not something
	// the user typed, so it fails rather than deleting something else.
	wrong := "web-1"
	require.Error(t, deleteComputeInstance(context.Background(), client, &wrong))
	assert.Len(t, client.deleted, 1)
}

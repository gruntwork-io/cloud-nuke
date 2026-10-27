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

// fakeCloudSQL answers with the instances a test names and records what it was
// asked to delete.
type fakeCloudSQL struct {
	instances []CloudSQLInstance
	listErr   error

	deleted []string
}

func (f *fakeCloudSQL) ListInstances(ctx context.Context, projectID string) ([]CloudSQLInstance, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.instances, nil
}

func (f *fakeCloudSQL) DeleteInstance(ctx context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func TestCloudSQLInstances_ResourceName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "cloud-sql-instance", NewCloudSQLInstances().ResourceName())
}

// A Cloud SQL instance name carries no location, so the region comes off the
// API's answer and this lister applies the filter. Without it, a run narrowed
// to one region deletes databases in every other one.
func TestListCloudSQLInstancesAppliesTheLocationFilter(t *testing.T) {
	t.Parallel()

	client := &fakeCloudSQL{instances: []CloudSQLInstance{
		{Name: "in-central", Region: "us-central1"},
		{Name: "in-west4", Region: "europe-west4"},
	}}

	for _, tc := range []struct {
		name     string
		cfg      GcpConfig
		expected []string
	}{
		{"no filter keeps every instance", GcpConfig{ProjectID: "p"}, []string{"in-central", "in-west4"}},
		{"a region keeps its own", GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}}, []string{"in-central"}},
		{"an exclusion drops its own", GcpConfig{ProjectID: "p", ExcludeLocations: []string{"us-central1"}}, []string{"in-west4"}},
		{"a region is matched however it is typed", GcpConfig{ProjectID: "p", Locations: []string{"US-CENTRAL1"}}, []string{"in-central"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identifiers, err := listCloudSQLInstances(context.Background(), client,
				resource.Scope{ProjectID: "p"}, config.ResourceType{}, tc.cfg)
			require.NoError(t, err)

			kept := make([]string, 0, len(identifiers))
			for _, id := range identifiers {
				kept = append(kept, *id)
			}
			assert.ElementsMatch(t, tc.expected, kept)
		})
	}
}

// A protected instance refuses its delete until somebody turns the protection
// off, which is the guard a team puts on a database it means to keep.
func TestListCloudSQLInstancesLeavesAProtectedInstanceAlone(t *testing.T) {
	t.Parallel()

	client := &fakeCloudSQL{instances: []CloudSQLInstance{
		{Name: "protected", Region: "us-central1", Protected: true},
		{Name: "ordinary", Region: "us-central1"},
	}}

	identifiers, err := listCloudSQLInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)

	require.Len(t, identifiers, 1)
	assert.Equal(t, "ordinary", *identifiers[0])
}

func TestListCloudSQLInstancesFiltersOnNameLabelsAndTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	client := &fakeCloudSQL{instances: []CloudSQLInstance{
		{Name: "keep-me", Region: "us-central1", Created: now, Labels: map[string]string{"env": "test"}},
		{Name: "drop-me", Region: "us-central1", Created: now},
	}}

	byName := config.ResourceType{IncludeRule: config.FilterRule{
		NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("^keep-")}},
	}}
	identifiers, err := listCloudSQLInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byName, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Equal(t, "keep-me", *identifiers[0])

	// A label rule reads the instance's user labels, which the lister has to
	// hand the config layer or a run narrowed by label deletes nothing.
	byLabel := config.ResourceType{IncludeRule: config.FilterRule{
		Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("^test$")}},
	}}
	identifiers, err = listCloudSQLInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byLabel, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Equal(t, "keep-me", *identifiers[0])

	cutoff := now.Add(-time.Hour)
	byTime := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: &cutoff}}
	identifiers, err = listCloudSQLInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byTime, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	assert.Empty(t, identifiers)
}

func TestListCloudSQLInstancesReportsAListFailure(t *testing.T) {
	t.Parallel()

	client := &fakeCloudSQL{listErr: errors.New("permission denied")}

	_, err := listCloudSQLInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestNukeCloudSQLInstanceDeletesByName(t *testing.T) {
	t.Parallel()

	client := &fakeCloudSQL{}
	name := "web-db"

	require.NoError(t, deleteCloudSQLInstance(context.Background(), client, &name))
	assert.Equal(t, []string{"web-db"}, client.deleted)
}

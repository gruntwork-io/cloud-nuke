package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
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

// Cloud SQL refuses to delete an instance that still has a replica, and the
// nuker works through the identifiers one at a time, so a primary listed ahead
// of its replica fails its delete outright.
func TestListCloudSQLInstancesReturnsReplicasBeforeTheirPrimary(t *testing.T) {
	t.Parallel()

	// The API answers in this order: the primary first, then a replica of it,
	// then a replica of that replica.
	client := &fakeCloudSQL{instances: []CloudSQLInstance{
		{Name: "primary", Region: "us-central1"},
		{Name: "replica", Region: "us-central1", MasterInstanceName: "a-project:primary"},
		{Name: "cascading", Region: "us-central1", MasterInstanceName: "replica"},
	}}

	identifiers, err := listCloudSQLInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)

	deleted := make([]string, 0, len(identifiers))
	for _, id := range identifiers {
		deleted = append(deleted, *id)
	}
	assert.Equal(t, []string{"cascading", "replica", "primary"}, deleted)
}

// Instances with no replication between them keep the order the API answered
// in, so this ordering does not reshuffle a run it has nothing to say about.
func TestListCloudSQLInstancesKeepsTheApiOrderAmongUnrelatedInstances(t *testing.T) {
	t.Parallel()

	// Enough instances that an unstable sort would be seen to shuffle the ones
	// it finds equal, and one replica so that the ordering runs at all.
	var instances []CloudSQLInstance
	var expected []string
	for i := 0; i < 40; i++ {
		name := fmt.Sprintf("db-%02d", (i*17)%40)
		instances = append(instances, CloudSQLInstance{Name: name, Region: "us-central1"})
		expected = append(expected, name)
	}
	instances = append(instances, CloudSQLInstance{Name: "db-replica", Region: "us-central1", MasterInstanceName: "db-00"})
	expected = append([]string{"db-replica"}, expected...)

	client := &fakeCloudSQL{instances: instances}

	identifiers, err := listCloudSQLInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)

	order := make([]string, 0, len(identifiers))
	for _, id := range identifiers {
		order = append(order, *id)
	}
	assert.Equal(t, expected, order)
}

// A replica a filter left out still sits between its primary and the top of the
// chain, so the depth has to be read over every instance in the project rather
// than over the ones this run deletes.
func TestListCloudSQLInstancesOrdersThroughAFilteredOutReplica(t *testing.T) {
	t.Parallel()

	client := &fakeCloudSQL{instances: []CloudSQLInstance{
		{Name: "keep-primary", Region: "us-central1"},
		{Name: "skip-replica", Region: "us-central1", MasterInstanceName: "keep-primary"},
		{Name: "keep-cascading", Region: "us-central1", MasterInstanceName: "skip-replica"},
	}}

	byName := config.ResourceType{IncludeRule: config.FilterRule{
		NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("^keep-")}},
	}}
	identifiers, err := listCloudSQLInstances(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byName, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)

	order := make([]string, 0, len(identifiers))
	for _, id := range identifiers {
		order = append(order, *id)
	}
	assert.Equal(t, []string{"keep-cascading", "keep-primary"}, order)
}

// A malformed answer describing a replication cycle would otherwise spin
// forever while the run looks for the top of the chain.
func TestReplicaDepthStopsOnACycle(t *testing.T) {
	t.Parallel()

	masterOf := map[string]string{"one": "two", "two": "one"}
	assert.Equal(t, 1, replicaDepth("one", masterOf))
}

// Cloud SQL reports an unreachable region as a warning on an answer that
// carries HTTP 200, so a run that read past it would report a clean sweep of a
// project whose missing region still holds databases.
func TestIncompleteListRefusesAnAnswerMissingInstances(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		warnings []*sqladmin.ApiWarning
		wants    string
	}{
		{
			name:     "an unreachable region",
			warnings: []*sqladmin.ApiWarning{{Code: "REGION_UNREACHABLE", Region: "europe-west4", Message: "region is not reachable"}},
			wants:    "europe-west4",
		},
		{
			name:     "more instances than the page holds",
			warnings: []*sqladmin.ApiWarning{{Code: "MAX_RESULTS_EXCEEDS_LIMIT", Message: "maxResults exceeds the limit"}},
			wants:    "MAX_RESULTS_EXCEEDS_LIMIT",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := incompleteList(tc.warnings)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wants)
		})
	}
}

// The other warnings say nothing about the answer being short, so refusing the
// run on one of them would fail a sweep that listed every instance.
func TestIncompleteListAllowsAWarningAboutSomethingElse(t *testing.T) {
	t.Parallel()

	assert.NoError(t, incompleteList(nil))
	assert.NoError(t, incompleteList([]*sqladmin.ApiWarning{nil}))
	assert.NoError(t, incompleteList([]*sqladmin.ApiWarning{{Code: "COMPROMISED_CREDENTIALS"}}))
	assert.NoError(t, incompleteList([]*sqladmin.ApiWarning{{Code: "INTERNAL_STATE_FAILURE"}}))
}

// A replica can sit in a region the run was narrowed away from, which leaves its
// primary's delete to fail for a reason the run otherwise never states.
//
// Not parallel: it swaps the writer pterm warns through, which every other test
// in this package shares.
func TestListCloudSQLInstancesSaysWhenAReplicaIsLeftBehind(t *testing.T) {
	client := &fakeCloudSQL{instances: []CloudSQLInstance{
		{Name: "primary", Region: "us-central1"},
		{Name: "far-replica", Region: "europe-west4", MasterInstanceName: "primary"},
	}}

	said := captureCloudSQLWarnings(t, func() {
		identifiers, err := listCloudSQLInstances(context.Background(), client,
			resource.Scope{ProjectID: "p"}, config.ResourceType{},
			GcpConfig{ProjectID: "p", Locations: []string{"us-central1"}})
		require.NoError(t, err)
		require.Len(t, identifiers, 1)
		assert.Equal(t, "primary", *identifiers[0])
	})
	assert.Contains(t, said, "far-replica")

	// With both in the run there is nothing to say: the ordering handles it.
	said = captureCloudSQLWarnings(t, func() {
		identifiers, err := listCloudSQLInstances(context.Background(), client,
			resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
		require.NoError(t, err)
		require.Len(t, identifiers, 2)
	})
	assert.NotContains(t, said, "far-replica")
}

// captureCloudSQLWarnings reads what the run told the engineer, which is the
// whole of what the case above changes.
func captureCloudSQLWarnings(t *testing.T, run func()) string {
	t.Helper()

	var said bytes.Buffer
	previous := pterm.Warning.Writer
	pterm.Warning.Writer = &said
	t.Cleanup(func() { pterm.Warning.Writer = previous })

	run()
	return said.String()
}

package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	cloudinv "github.com/james00012/cloud-inventory"
	gcpinv "github.com/james00012/cloud-inventory/gcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runv2 "google.golang.org/api/run/v2"
)

// fakeCloudRun stands in for the shared inventory. It records the scope it was
// handed, which is how this file checks that --region reaches the library as
// the regions to ask rather than as answers to drop.
type fakeCloudRun struct {
	services   []gcpinv.CloudRunService
	incomplete cloudinv.Incomplete
	listErr    error

	scope   gcpinv.Scope
	deleted []cloudinv.ID
}

func (f *fakeCloudRun) List(ctx context.Context, scope gcpinv.Scope) ([]gcpinv.CloudRunService, cloudinv.Incomplete, error) {
	f.scope = scope
	if f.listErr != nil {
		return nil, cloudinv.Incomplete{}, f.listErr
	}
	return f.services, f.incomplete, nil
}

func (f *fakeCloudRun) Delete(ctx context.Context, id cloudinv.ID) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func serviceIn(location, name string) gcpinv.CloudRunService {
	full := "projects/p/locations/" + location + "/services/" + name
	return gcpinv.CloudRunService{
		GoogleCloudRunV2Service: &runv2.GoogleCloudRunV2Service{Name: full},
		ID:                      cloudinv.ID(full),
		Location:                location,
	}
}

func TestCloudRunServices_ResourceName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "cloud-run-service", NewCloudRunServices().ResourceName())
}

// --region has to reach the library as the regions to ask. Handing it nothing
// and dropping answers afterwards would spend a call per region Cloud Run
// offers on a run narrowed to one.
func TestListCloudRunServicesAsksTheLibraryForOnlyTheRegionsRequested(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{}

	_, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{},
		GcpConfig{ProjectID: "p", Locations: []string{"europe-west4"}})
	require.NoError(t, err)

	assert.Equal(t, "p", client.scope.ProjectID)
	assert.Equal(t, []string{"europe-west4"}, client.scope.Locations)
}

// An exclusion cannot be handed over as a list of regions to ask, so this
// resource still reads it against what came back.
func TestListCloudRunServicesAppliesAnExclusionItself(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{services: []gcpinv.CloudRunService{
		serviceIn("us-central1", "one"),
		serviceIn("europe-west4", "two"),
	}}

	identifiers, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{},
		GcpConfig{ProjectID: "p", ExcludeLocations: []string{"europe-west4"}})
	require.NoError(t, err)

	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/us-central1/services/one", *identifiers[0])
}

// Every region refusing is not a project with no services, and reporting zero
// would call the project clean.
func TestListCloudRunServicesFailsWhenEveryRegionRefused(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{incomplete: cloudinv.Incomplete{
		Asked: 2,
		Gaps: []cloudinv.Gap{
			{Scope: "us-central1", Fault: cloudinv.FaultDenied},
			{Scope: "europe-west4", Fault: cloudinv.FaultDenied},
		},
	}}

	_, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "any of the 2 regions")
}

// One region of several refusing costs that region alone.
func TestListCloudRunServicesKeepsWhatOneRefusedRegionLeaves(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{
		services: []gcpinv.CloudRunService{serviceIn("europe-west4", "two")},
		incomplete: cloudinv.Incomplete{
			Asked: 2,
			Gaps:  []cloudinv.Gap{{Scope: "us-central1", Fault: cloudinv.FaultDenied}},
		},
	}

	identifiers, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
}

func TestListCloudRunServicesReportsAListFailure(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{listErr: errors.New("run.googleapis.com is not enabled")}

	_, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, config.ResourceType{}, GcpConfig{ProjectID: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not enabled")
}

// A config rule is written against the name the console shows, while the
// identifier stays the full name the delete takes.
func TestListCloudRunServicesFiltersOnTheShortNameAndKeepsTheFullOne(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{services: []gcpinv.CloudRunService{
		serviceIn("us-central1", "keep-me"),
		serviceIn("us-central1", "drop-me"),
	}}

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

	client := &fakeCloudRun{services: []gcpinv.CloudRunService{labelled, plain}}

	byLabel := config.ResourceType{IncludeRule: config.FilterRule{
		Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("^test$")}},
	}}
	identifiers, err := listCloudRunServices(context.Background(), client,
		resource.Scope{ProjectID: "p"}, byLabel, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/us-central1/services/keep-me", *identifiers[0])

	// A time rule has to keep the service it does not name. Asserting only that
	// the recent one goes would pass just as well if the lister never handed
	// the creation time over at all.
	old := serviceIn("us-central1", "old-one")
	old.Created = now.Add(-48 * time.Hour)
	aged := &fakeCloudRun{services: []gcpinv.CloudRunService{labelled, old}}

	cutoff := now.Add(-time.Hour)
	byTime := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: &cutoff}}
	identifiers, err = listCloudRunServices(context.Background(), aged,
		resource.Scope{ProjectID: "p"}, byTime, GcpConfig{ProjectID: "p"})
	require.NoError(t, err)
	require.Len(t, identifiers, 1)
	assert.Equal(t, "projects/p/locations/us-central1/services/old-one", *identifiers[0])
}

func TestNukeCloudRunServiceDeletesByFullName(t *testing.T) {
	t.Parallel()

	client := &fakeCloudRun{}
	name := "projects/p/locations/us-central1/services/one"

	require.NoError(t, deleteCloudRunService(context.Background(), client, &name))
	assert.Equal(t, []cloudinv.ID{cloudinv.ID(name)}, client.deleted)
}

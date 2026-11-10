package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCloudFunctions answers with the functions a test names and records what
// it was asked to delete.
type fakeCloudFunctions struct {
	functions  []CloudFunction
	listErr    error
	parentSeen string
	deleted    []string
}

func (f *fakeCloudFunctions) ListFunctions(ctx context.Context, parent string) ([]CloudFunction, error) {
	f.parentSeen = parent
	return f.functions, f.listErr
}

func (f *fakeCloudFunctions) DeleteFunction(ctx context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func TestCloudFunctions_ResourceName(t *testing.T) {
	assert.Equal(t, "cloud-function", NewCloudFunctions().ResourceName())
}

func TestCloudFunctions_MaxBatchSize(t *testing.T) {
	assert.Equal(t, 50, NewCloudFunctions().MaxBatchSize())
}

func TestListCloudFunctionsAsksEveryLocationAtOnce(t *testing.T) {
	t.Parallel()

	client := &fakeCloudFunctions{}
	_, err := listCloudFunctions(context.Background(), client,
		resource.Scope{ProjectID: "my-project"}, config.ResourceType{})
	require.NoError(t, err)

	// The wildcard location is what makes one call cover every region, and it
	// is why this resource needs no location fan-out.
	assert.Equal(t, "projects/my-project/locations/-", client.parentSeen)
}

func TestListCloudFunctionsFilters(t *testing.T) {
	t.Parallel()

	now := time.Now()
	old := now.Add(-48 * time.Hour)
	central := "projects/p/locations/us-central1/functions/keep-me"
	west := "projects/p/locations/europe-west4/functions/drop-me"

	for _, tc := range []struct {
		name     string
		cfg      config.ResourceType
		expected []string
	}{
		{
			name:     "no filter keeps every function",
			cfg:      config.ResourceType{},
			expected: []string{central, west},
		},
		{
			name: "a name filter matches the whole resource name",
			cfg: config.ResourceType{IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("keep-me$")}},
			}},
			expected: []string{central},
		},
		{
			name: "a time filter reads the function's update time",
			cfg: config.ResourceType{ExcludeRule: config.FilterRule{
				TimeAfter: &old,
			}},
			expected: []string{west},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &fakeCloudFunctions{functions: []CloudFunction{
				{Name: central, Updated: now},
				{Name: west, Updated: old.Add(-time.Hour)},
			}}

			identifiers, err := listCloudFunctions(context.Background(), client,
				resource.Scope{ProjectID: "p"}, tc.cfg)
			require.NoError(t, err)

			var names []string
			for _, identifier := range identifiers {
				names = append(names, *identifier)
			}
			assert.Equal(t, tc.expected, names)
		})
	}
}

func TestCloudFunctionsHonorsTheLocationFilterThroughTheAdapter(t *testing.T) {
	t.Parallel()

	central := "projects/p/locations/us-central1/functions/one"
	west := "projects/p/locations/europe-west4/functions/two"

	// The adapter filters on the location in each name, so this resource needs
	// no location code of its own. Asking for one location must reach only it.
	r := NewCloudFunctions()

	adapter, ok := r.(*GcpResourceAdapter[CloudFunctionsAPI])
	require.True(t, ok)

	// Hand the fake over in place of the client the real initializer builds,
	// which is what the narrow interface is for. Replacing it after Init would
	// not do: Init needs Application Default Credentials to reach the SDK, and
	// the error it records when there are none is never cleared, so this test
	// would depend on whose machine it runs on.
	adapter.InitClient = WrapGcpInitClient(func(res *resource.Resource[CloudFunctionsAPI], cfg GcpConfig) {
		res.Scope.ProjectID = cfg.ProjectID
		res.Client = &fakeCloudFunctions{functions: []CloudFunction{
			{Name: central, Updated: time.Now()},
			{Name: west, Updated: time.Now()},
		}}
	})
	adapter.Init(GcpConfig{ProjectID: "p", Locations: []string{"europe-west4"}})

	identifiers, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.NoError(t, err)
	assert.Equal(t, []string{west}, identifiers)
}

func TestNukeCloudFunctionsDeletesByName(t *testing.T) {
	t.Parallel()

	client := &fakeCloudFunctions{}
	name := "projects/p/locations/us-central1/functions/one"
	require.NoError(t, deleteCloudFunction(context.Background(), client, &name))
	assert.Equal(t, []string{name}, client.deleted)
}

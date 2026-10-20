package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeResourceGroups answers with the groups a test names and records what it
// was asked to delete.
type fakeResourceGroups struct {
	groups  []ResourceGroup
	listErr error
	deleted []string
}

func (f *fakeResourceGroups) ListResourceGroups(ctx context.Context) ([]ResourceGroup, error) {
	return f.groups, f.listErr
}

func (f *fakeResourceGroups) DeleteResourceGroup(ctx context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func TestResourceGroups_ResourceName(t *testing.T) {
	assert.Equal(t, "resource-group", NewResourceGroups().ResourceName())
}

func TestResourceGroups_MaxBatchSize(t *testing.T) {
	assert.Equal(t, 50, NewResourceGroups().MaxBatchSize())
}

func TestListResourceGroupsLeavesAloneWhatAzureWillNotLetUsDelete(t *testing.T) {
	t.Parallel()

	client := &fakeResourceGroups{groups: []ResourceGroup{
		{Name: "mine", Location: "westeurope"},
		{Name: "MC_aks_cluster_westeurope", Location: "westeurope", Managed: true},
		{Name: "locked", Location: "westeurope", Locked: true},
	}}

	identifiers, err := listResourceGroups(context.Background(), client,
		config.ResourceType{}, AzureConfig{SubscriptionID: "sub"})
	require.NoError(t, err)

	// A managed group goes when its owning service goes, and a locked group's
	// delete is refused by the API, so neither is worth attempting.
	require.Len(t, identifiers, 1)
	assert.Equal(t, "mine", *identifiers[0])
}

func TestListResourceGroupsAppliesTheLocationFilter(t *testing.T) {
	t.Parallel()

	groups := []ResourceGroup{
		{Name: "in-west", Location: "westeurope"},
		{Name: "in-east", Location: "eastus"},
	}

	for _, tc := range []struct {
		name     string
		cfg      AzureConfig
		expected []string
	}{
		{
			name:     "no filter keeps every group",
			cfg:      AzureConfig{SubscriptionID: "sub"},
			expected: []string{"in-west", "in-east"},
		},
		{
			name:     "an included location keeps only its own",
			cfg:      AzureConfig{SubscriptionID: "sub", Locations: []string{"westeurope"}},
			expected: []string{"in-west"},
		},
		{
			name:     "an excluded location drops its own",
			cfg:      AzureConfig{SubscriptionID: "sub", ExcludeLocations: []string{"westeurope"}},
			expected: []string{"in-east"},
		},
		{
			name: "a location typed the way the portal shows it still matches",
			// The portal writes "West Europe" and the API answers "westeurope".
			cfg:      AzureConfig{SubscriptionID: "sub", Locations: []string{"West Europe"}},
			expected: []string{"in-west"},
		},
		{
			name:     "an exclusion beats an inclusion",
			cfg:      AzureConfig{SubscriptionID: "sub", Locations: []string{"westeurope"}, ExcludeLocations: []string{"westeurope"}},
			expected: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identifiers, err := listResourceGroups(context.Background(),
				&fakeResourceGroups{groups: groups}, config.ResourceType{}, tc.cfg)
			require.NoError(t, err)

			var names []string
			for _, identifier := range identifiers {
				names = append(names, *identifier)
			}
			assert.Equal(t, tc.expected, names)
		})
	}
}

func TestListResourceGroupsFiltersOnNameTagsAndTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	client := &fakeResourceGroups{groups: []ResourceGroup{
		{Name: "keep-me", Location: "westeurope", Created: now, Tags: map[string]string{"env": "test"}},
		{Name: "drop-me", Location: "westeurope", Created: now},
	}}

	cfg := config.ResourceType{IncludeRule: config.FilterRule{
		NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("^keep-")}},
	}}

	identifiers, err := listResourceGroups(context.Background(), client, cfg, AzureConfig{SubscriptionID: "sub"})
	require.NoError(t, err)

	require.Len(t, identifiers, 1)
	assert.Equal(t, "keep-me", *identifiers[0])
}

func TestListResourceGroupsOmitsATimeItWasNotGiven(t *testing.T) {
	t.Parallel()

	// Azure reports no creation time unless the caller asks for the field, and
	// the zero time must not read as "created in 1970", which an --older-than
	// filter would then delete.
	yesterday := time.Now().Add(-24 * time.Hour)
	client := &fakeResourceGroups{groups: []ResourceGroup{{Name: "undated", Location: "westeurope"}}}

	identifiers, err := listResourceGroups(context.Background(), client,
		config.ResourceType{IncludeRule: config.FilterRule{TimeBefore: &yesterday}},
		AzureConfig{SubscriptionID: "sub"})
	require.NoError(t, err)

	assert.Empty(t, identifiers, "a group with no reported creation time is not matched by a time filter")
}

func TestListResourceGroupsReportsAListFailure(t *testing.T) {
	t.Parallel()

	client := &fakeResourceGroups{listErr: errors.New("the subscription was not found")}
	_, err := listResourceGroups(context.Background(), client, config.ResourceType{}, AzureConfig{SubscriptionID: "sub"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "the subscription was not found")
}

func TestNukeResourceGroupDeletesByName(t *testing.T) {
	t.Parallel()

	client := &fakeResourceGroups{}
	name := "mine"
	require.NoError(t, deleteResourceGroup(context.Background(), client, &name))
	assert.Equal(t, []string{name}, client.deleted)
}

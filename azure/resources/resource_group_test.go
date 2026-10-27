package resources

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/pterm/pterm"

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

func TestResourceGroupFromIDReadsTheGroupAtEveryLockScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "lock on the group itself",
			id:   "/subscriptions/s/resourceGroups/MyGroup/providers/Microsoft.Authorization/locks/no-delete",
			want: "mygroup",
		},
		{
			name: "lock on a resource inside the group, which still fails the group delete",
			id:   "/subscriptions/s/resourceGroups/MyGroup/providers/Microsoft.Storage/storageAccounts/a/providers/Microsoft.Authorization/locks/no-delete",
			want: "mygroup",
		},
		{
			name: "lower-cased spelling Azure also answers with",
			id:   "/subscriptions/s/resourcegroups/MyGroup/providers/Microsoft.Authorization/locks/no-delete",
			want: "mygroup",
		},
		{
			name: "subscription-scoped lock names no group",
			id:   "/subscriptions/s/providers/Microsoft.Authorization/locks/no-delete",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, resourceGroupFromID(tc.id))
		})
	}
}

// Azure reports no creation time for a resource group, so a run carrying a
// time filter, which every scheduled sweep does, can delete none of them. The
// lister leaves them alone rather than deleting something whose age it cannot
// read, and says how many and why, since a sweep that silently finds nothing
// reads as a subscription that is already clean.
// This one does not run in parallel: it reads what the run printed, and the
// printer is one package-level value every test in this file writes through.
func TestListResourceGroupsSaysWhenItCannotReadAGroupsAge(t *testing.T) {
	client := &fakeResourceGroups{groups: []ResourceGroup{
		{Name: "no-created-time", Location: "westeurope"},
		{Name: "also-no-created-time", Location: "westeurope"},
	}}

	cutoff := time.Now().Add(-2 * time.Hour)
	filtered := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: &cutoff}}

	said := captureWarnings(t, func() {
		identifiers, err := listResourceGroups(context.Background(), client, filtered, AzureConfig{SubscriptionID: "sub"})
		require.NoError(t, err)
		assert.Empty(t, identifiers, "a group whose age Azure does not report is not deleted under a time filter")
	})
	assert.Contains(t, said, "Leaving 2 resource groups alone")
	assert.Contains(t, said, "does not report when a resource group was created")

	// Without a time filter the same groups are deletable, and nothing is said.
	said = captureWarnings(t, func() {
		unfiltered, err := listResourceGroups(context.Background(), client, config.ResourceType{}, AzureConfig{SubscriptionID: "sub"})
		require.NoError(t, err)
		assert.Len(t, unfiltered, 2)
	})
	assert.NotContains(t, said, "Leaving")
}

// captureWarnings reads what the run told the engineer, which is the whole of
// what the case above changes: the config layer already excludes a resource
// whose age is unknown, silently.
func captureWarnings(t *testing.T, run func()) string {
	t.Helper()

	var said bytes.Buffer
	previous := pterm.Warning.Writer
	pterm.Warning.Writer = &said
	t.Cleanup(func() { pterm.Warning.Writer = previous })

	run()
	return said.String()
}

func TestASubscriptionLockLocksEveryGroupInIt(t *testing.T) {
	t.Parallel()

	assert.True(t, isSubscriptionLockID("/subscriptions/s/providers/Microsoft.Authorization/locks/no-delete"))
	assert.True(t, isSubscriptionLockID("/subscriptions/s/providers/microsoft.authorization/locks/no-delete"))

	// A lock on a group, or on a resource inside one, names that group and is
	// read by resourceGroupFromID instead.
	assert.False(t, isSubscriptionLockID("/subscriptions/s/resourceGroups/g/providers/Microsoft.Authorization/locks/no-delete"))
	// An ID shape this code does not know must not lock the whole subscription.
	assert.False(t, isSubscriptionLockID("/subscriptions/s/providers/Microsoft.Compute/virtualMachines/vm"))
	assert.False(t, isSubscriptionLockID("/subscriptions/s"))
}

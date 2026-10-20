package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryValidate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		query     Query
		wantError string
		wantAfter []string
	}{
		{
			name:      "a subscription is required, since every ARM call takes one",
			query:     Query{},
			wantError: "--subscription-id is required",
		},
		{
			name:      "an empty subscription is not a subscription",
			query:     Query{SubscriptionIDs: []string{""}},
			wantError: "--subscription-id was empty",
		},
		{
			name:      "the same subscription twice would scan and delete it twice",
			query:     Query{SubscriptionIDs: []string{"sub-a", "sub-a"}},
			wantError: "was given twice",
		},
		{
			name:      "global is an AWS pseudo-region and matches no Azure resource here",
			query:     Query{SubscriptionIDs: []string{"sub"}, Locations: []string{"global"}},
			wantError: `"global"`,
		},
		{
			name:      "an empty location would filter every resource out",
			query:     Query{SubscriptionIDs: []string{"sub"}, ExcludeLocations: []string{" "}},
			wantError: "one was empty",
		},
		{
			name:      "an include list its own exclusions empty would scan everything",
			query:     Query{SubscriptionIDs: []string{"sub"}, Locations: []string{"westeurope"}, ExcludeLocations: []string{"westeurope"}},
			wantError: "every --region was excluded",
		},
		{
			name:      "negative parallelism is not a number of workers",
			query:     Query{SubscriptionIDs: []string{"sub"}, Parallelism: -1},
			wantError: "--parallelism must be >= 0",
		},
		{
			name:      "no location at all means the whole subscription",
			query:     Query{SubscriptionIDs: []string{"sub"}},
			wantAfter: nil,
		},
		{
			name:      "an exclusion narrows the include list it does not empty",
			query:     Query{SubscriptionIDs: []string{"sub"}, Locations: []string{"westeurope", "eastus"}, ExcludeLocations: []string{"eastus"}},
			wantAfter: []string{"westeurope"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			query := tc.query
			err := query.Validate()

			if tc.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantAfter, query.Locations)
		})
	}
}

func TestIsNukeable(t *testing.T) {
	t.Parallel()

	assert.True(t, IsNukeable("resource-group", nil, nil))
	assert.True(t, IsNukeable("resource-group", []string{"all"}, nil))
	assert.True(t, IsNukeable("resource-group", []string{"resource-group"}, nil))
	assert.False(t, IsNukeable("resource-group", []string{"something-else"}, nil))
	assert.False(t, IsNukeable("resource-group", nil, []string{"resource-group"}))
	assert.False(t, IsNukeable("resource-group", []string{"all"}, []string{"resource-group"}))
}

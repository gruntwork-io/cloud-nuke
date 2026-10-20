package gcp

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
			name:      "a project is required, since every GCP list call takes one",
			query:     Query{},
			wantError: "--project-id is required",
		},
		{
			name:      "global is an AWS pseudo-region and no GCP location",
			query:     Query{ProjectID: "p", Locations: []string{"global"}},
			wantError: `GCP has no "global" location`,
		},
		{
			name:      "global is refused whatever its case, since a runbook may shout it",
			query:     Query{ProjectID: "p", ExcludeLocations: []string{"GLOBAL"}},
			wantError: `GCP has no "global" location`,
		},
		{
			name:      "an empty location would filter every resource out",
			query:     Query{ProjectID: "p", Locations: []string{""}},
			wantError: "one was empty",
		},
		{
			name:      "an include list its own exclusions empty would scan everything",
			query:     Query{ProjectID: "p", Locations: []string{"us-central1"}, ExcludeLocations: []string{"us-central1"}},
			wantError: "every --region was excluded",
		},
		{
			name:      "negative parallelism is not a number of workers",
			query:     Query{ProjectID: "p", Parallelism: -1},
			wantError: "--parallelism must be >= 0",
		},
		{
			name:      "no location at all means the whole project",
			query:     Query{ProjectID: "p"},
			wantAfter: nil,
		},
		{
			name:      "an exclusion alone leaves the include list empty",
			query:     Query{ProjectID: "p", ExcludeLocations: []string{"europe-west1"}},
			wantAfter: nil,
		},
		{
			name:      "an exclusion narrows the include list it does not empty",
			query:     Query{ProjectID: "p", Locations: []string{"us-central1", "europe-west1"}, ExcludeLocations: []string{"europe-west1"}},
			wantAfter: []string{"us-central1"},
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

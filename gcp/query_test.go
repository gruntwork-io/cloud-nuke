package gcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_MissingProjectID(t *testing.T) {
	t.Parallel()
	q := &Query{}
	err := q.Validate(context.Background())
	assert.ErrorContains(t, err, "--project-id is required")
}

func TestValidateLocations_FiltersExcludedFromIncluded(t *testing.T) {
	t.Parallel()
	q := &Query{
		Locations:        []string{"us-central1", "us-east1"},
		ExcludeLocations: []string{"us-east1"},
	}
	require.NoError(t, q.validateLocations())
	assert.Equal(t, []string{"us-central1"}, q.Locations)
}

func TestValidateLocations_ErrorWhenAllExcluded(t *testing.T) {
	t.Parallel()
	q := &Query{
		Locations:        []string{"us-central1"},
		ExcludeLocations: []string{"us-central1"},
	}
	err := q.validateLocations()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to scan")
}

func TestValidateLocations_ExcludeWithoutIncludeIsValid(t *testing.T) {
	t.Parallel()
	q := &Query{
		ExcludeLocations: []string{"us-central1"},
	}
	require.NoError(t, q.validateLocations())
}

func TestValidateLocations_CaseInsensitiveExclusion(t *testing.T) {
	t.Parallel()
	q := &Query{
		Locations:        []string{"us-central1"},
		ExcludeLocations: []string{"US-CENTRAL1"},
	}
	err := q.validateLocations()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to scan")
}

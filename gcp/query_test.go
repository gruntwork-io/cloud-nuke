package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate_MissingProjectID(t *testing.T) {
	q := &Query{}
	err := q.Validate()
	assert.ErrorContains(t, err, "--project-id is required")
}

func TestValidateRegions_EmptyDefaultsToGlobal(t *testing.T) {
	q := &Query{Regions: []string{}}
	err := q.validateRegions()
	assert.NoError(t, err)
	assert.Equal(t, []string{GlobalRegion}, q.Regions)
}

func TestValidateRegions_PreservesExplicitRegions(t *testing.T) {
	q := &Query{Regions: []string{"us-central1", "us-west1"}}
	err := q.validateRegions()
	assert.NoError(t, err)
	assert.Equal(t, []string{"us-central1", "us-west1"}, q.Regions)
}

func TestValidateRegions_PartialExclusion(t *testing.T) {
	q := &Query{
		Regions:        []string{"us-central1", "us-west1", "europe-west1"},
		ExcludeRegions: []string{"us-west1"},
	}
	err := q.validateRegions()
	assert.NoError(t, err)
	assert.Equal(t, []string{"us-central1", "europe-west1"}, q.Regions)
}

func TestValidateRegions_ExcludeAllRegions(t *testing.T) {
	q := &Query{
		Regions:        []string{"us-central1"},
		ExcludeRegions: []string{"us-central1"},
	}
	err := q.validateRegions()
	assert.ErrorContains(t, err, "no regions to process")
}

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEC2PlacementGroups_ResourceName(t *testing.T) {
	r := NewEC2PlacementGroups()
	assert.Equal(t, "ec2-placement-groups", r.ResourceName())
}

func TestEC2PlacementGroups_MaxBatchSize(t *testing.T) {
	r := NewEC2PlacementGroups()
	assert.Equal(t, 200, r.MaxBatchSize())
}

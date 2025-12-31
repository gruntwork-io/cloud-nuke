package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudWatchLogGroups_ResourceName(t *testing.T) {
	r := NewCloudWatchLogGroups()
	assert.Equal(t, "cloudwatch-loggroup", r.ResourceName())
}

func TestCloudWatchLogGroups_MaxBatchSize(t *testing.T) {
	r := NewCloudWatchLogGroups()
	assert.Equal(t, 35, r.MaxBatchSize())
}

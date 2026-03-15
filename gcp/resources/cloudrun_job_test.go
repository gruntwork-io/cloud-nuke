package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudRunJobs_ResourceName(t *testing.T) {
	j := NewCloudRunJobs()
	assert.Equal(t, "cloud-run-job", j.ResourceName())
}

func TestCloudRunJobs_MaxBatchSize(t *testing.T) {
	j := NewCloudRunJobs()
	assert.Equal(t, 50, j.MaxBatchSize())
}

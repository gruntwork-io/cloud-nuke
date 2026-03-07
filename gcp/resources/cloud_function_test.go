package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudFunctions_ResourceName(t *testing.T) {
	cf := NewCloudFunctions()
	assert.Equal(t, "cloud-function", cf.ResourceName())
}

func TestCloudFunctions_MaxBatchSize(t *testing.T) {
	cf := NewCloudFunctions()
	assert.Equal(t, 50, cf.MaxBatchSize())
}

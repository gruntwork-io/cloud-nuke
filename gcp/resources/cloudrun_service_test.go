package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudRunServices_ResourceName(t *testing.T) {
	svc := NewCloudRunServices()
	assert.Equal(t, "cloud-run-service", svc.ResourceName())
}

func TestCloudRunServices_MaxBatchSize(t *testing.T) {
	svc := NewCloudRunServices()
	assert.Equal(t, 50, svc.MaxBatchSize())
}

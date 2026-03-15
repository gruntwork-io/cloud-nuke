package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudSQLInstances_ResourceName(t *testing.T) {
	cs := NewCloudSQLInstances()
	assert.Equal(t, "cloud-sql-instance", cs.ResourceName())
}

func TestCloudSQLInstances_MaxBatchSize(t *testing.T) {
	cs := NewCloudSQLInstances()
	assert.Equal(t, 50, cs.MaxBatchSize())
}

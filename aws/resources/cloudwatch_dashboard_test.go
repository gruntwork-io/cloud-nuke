package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudWatchDashboards_ResourceName(t *testing.T) {
	r := NewCloudWatchDashboards()
	assert.Equal(t, "cloudwatch-dashboard", r.ResourceName())
}

func TestCloudWatchDashboards_MaxBatchSize(t *testing.T) {
	r := NewCloudWatchDashboards()
	assert.Equal(t, 49, r.MaxBatchSize())
}

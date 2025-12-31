package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLaunchConfigs_ResourceName(t *testing.T) {
	r := NewLaunchConfigs()
	assert.Equal(t, "lc", r.ResourceName())
}

func TestLaunchConfigs_MaxBatchSize(t *testing.T) {
	r := NewLaunchConfigs()
	assert.Equal(t, 49, r.MaxBatchSize())
}

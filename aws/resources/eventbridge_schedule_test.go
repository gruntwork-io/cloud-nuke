package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventBridgeSchedule_ResourceName(t *testing.T) {
	r := NewEventBridgeSchedule()
	assert.Equal(t, "event-bridge-schedule", r.ResourceName())
}

func TestEventBridgeSchedule_MaxBatchSize(t *testing.T) {
	r := NewEventBridgeSchedule()
	assert.Equal(t, 100, r.MaxBatchSize())
}

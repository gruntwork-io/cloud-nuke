package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPubSubTopics_ResourceName(t *testing.T) {
	ps := NewPubSubTopics()
	assert.Equal(t, "gcp-pubsub-topic", ps.ResourceName())
}

func TestPubSubTopics_MaxBatchSize(t *testing.T) {
	ps := NewPubSubTopics()
	assert.Equal(t, 50, ps.MaxBatchSize())
}

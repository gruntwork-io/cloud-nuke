package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKinesisStreams_ResourceName(t *testing.T) {
	r := NewKinesisStreams()
	assert.Equal(t, "kinesis-stream", r.ResourceName())
}

func TestKinesisStreams_MaxBatchSize(t *testing.T) {
	r := NewKinesisStreams()
	assert.Equal(t, 35, r.MaxBatchSize())
}

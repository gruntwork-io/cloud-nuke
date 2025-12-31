package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGCSBuckets_ResourceName(t *testing.T) {
	gcs := NewGCSBuckets()
	assert.Equal(t, "gcs-bucket", gcs.ResourceName())
}

func TestGCSBuckets_MaxBatchSize(t *testing.T) {
	gcs := NewGCSBuckets()
	assert.Equal(t, 50, gcs.MaxBatchSize())
}

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKinesisFirehose_ResourceName(t *testing.T) {
	r := NewKinesisFirehose()
	assert.Equal(t, "kinesis-firehose", r.ResourceName())
}

func TestKinesisFirehose_MaxBatchSize(t *testing.T) {
	r := NewKinesisFirehose()
	assert.Equal(t, 35, r.MaxBatchSize())
}

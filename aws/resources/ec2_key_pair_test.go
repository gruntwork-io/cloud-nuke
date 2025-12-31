package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEC2KeyPairs_ResourceName(t *testing.T) {
	r := NewEC2KeyPairs()
	assert.Equal(t, "ec2-keypairs", r.ResourceName())
}

func TestEC2KeyPairs_MaxBatchSize(t *testing.T) {
	r := NewEC2KeyPairs()
	assert.Equal(t, 200, r.MaxBatchSize())
}

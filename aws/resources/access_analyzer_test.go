package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessAnalyzer_ResourceName(t *testing.T) {
	r := NewAccessAnalyzer()
	assert.Equal(t, "accessanalyzer", r.ResourceName())
}

func TestAccessAnalyzer_MaxBatchSize(t *testing.T) {
	r := NewAccessAnalyzer()
	assert.Equal(t, 10, r.MaxBatchSize())
}

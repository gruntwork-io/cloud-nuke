package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitEmpty(t *testing.T) {
	t.Parallel()

	array := []string{}
	batches := split(array, 2)

	assert.Len(t, batches, 0)
}

func TestSplit(t *testing.T) {
	t.Parallel()

	array := []string{"a", "b", "c", "d"}
	batches := split(array, 2)

	assert.Len(t, batches, 2)
	assert.Equal(t, "a", batches[0][0])
}

func TestSplitLarge(t *testing.T) {
	t.Parallel()

	array := []string{"a", "b", "c", "d"}
	batches := split(array, 5)

	assert.Len(t, batches, 1)
	assert.Equal(t, "a", batches[0][0])
}

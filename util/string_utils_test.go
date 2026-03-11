package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello", Truncate("hello", 10))
	assert.Equal(t, "hello", Truncate("hello", 5))
	assert.Equal(t, "hello", Truncate("hello world", 5))
	assert.Equal(t, "", Truncate("", 10))
}

func TestRemoveNewlines(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello world", RemoveNewlines("hello\nworld"))
	assert.Equal(t, "a b c", RemoveNewlines("a\nb\nc"))
	assert.Equal(t, "", RemoveNewlines(""))
}

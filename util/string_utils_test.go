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

func TestDerefString(t *testing.T) {
	t.Parallel()
	s := "hello"
	assert.Equal(t, "hello", DerefString(&s))
	assert.Equal(t, "", DerefString(nil))
}

func TestDerefStringSlice(t *testing.T) {
	t.Parallel()
	a, b := "a", "b"
	assert.Equal(t, []string{"a", "b"}, DerefStringSlice([]*string{&a, &b}))
	assert.Equal(t, []string{"a", ""}, DerefStringSlice([]*string{&a, nil}))
	assert.Equal(t, []string{}, DerefStringSlice([]*string{}))
}

func TestDifference_NilSafe(t *testing.T) {
	t.Parallel()
	a, b, c := "a", "b", "c"

	// Normal case
	diff := Difference([]*string{&a, &b, &c}, []*string{&b})
	assert.Len(t, diff, 2)

	// Nil elements in slices should not panic
	diff = Difference([]*string{&a, nil, &c}, []*string{&b, nil})
	assert.Len(t, diff, 2) // "a" and "c" (nil is skipped)
}

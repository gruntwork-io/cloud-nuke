package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"shorter than limit", "hello", 10, "hello"},
		{"exactly at limit", "hello", 5, "hello"},
		{"over limit", "hello world", 5, "hello"},
		{"empty string", "", 10, ""},
		{"zero limit", "hello", 0, ""},
		{"single char over", "ab", 1, "a"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Truncate(tc.input, tc.maxLen))
		})
	}
}

func TestRemoveNewlines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no newlines", "hello world", "hello world"},
		{"single newline", "hello\nworld", "hello world"},
		{"multiple newlines", "a\nb\nc", "a b c"},
		{"only newlines", "\n\n\n", "   "},
		{"empty string", "", ""},
		{"trailing newline", "hello\n", "hello "},
		{"leading newline", "\nhello", " hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, RemoveNewlines(tc.input))
		})
	}
}

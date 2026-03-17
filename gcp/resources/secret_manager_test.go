package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractLocationFromResourceName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"global secret", "projects/my-project/secrets/my-secret", ""},
		{"regional secret", "projects/my-project/locations/us-central1/secrets/my-secret", "us-central1"},
		{"locations segment with no value", "projects/my-project/locations", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, ExtractLocationFromResourceName(tc.input))
		})
	}
}

func TestShouldIncludeGlobalEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		locs     []string
		exclude  []string
		expected bool
	}{
		{"no locations includes global", nil, nil, true},
		{"global excluded", nil, []string{"global"}, false},
		{"specific location excludes global", []string{"us-central1"}, nil, false},
		{"global in locations", []string{"global", "us-central1"}, nil, true},
		{"global in both includes and excludes", []string{"global"}, []string{"global"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, shouldIncludeGlobalEndpoint(tc.locs, tc.exclude))
		})
	}
}

func TestMatchesLocationFilter(t *testing.T) {
	t.Parallel()

	// No filter → match everything
	assert.True(t, MatchesLocationFilter("us-central1", nil, nil))

	// Include filter
	assert.True(t, MatchesLocationFilter("us-central1", []string{"us-central1"}, nil))
	assert.False(t, MatchesLocationFilter("us-east1", []string{"us-central1"}, nil))

	// Exclude filter
	assert.False(t, MatchesLocationFilter("us-central1", nil, []string{"us-central1"}))

	// Exclude takes precedence over include
	assert.False(t, MatchesLocationFilter("us-central1", []string{"us-central1"}, []string{"us-central1"}))

	// Case insensitive
	assert.True(t, MatchesLocationFilter("US-CENTRAL1", []string{"us-central1"}, nil))
}

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testResourceTypes = []string{"alpha", "beta", "gamma"}

func TestIsNukeable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		resourceType  string
		resourceTypes []string
		expected      bool
	}{
		{"nil list means all", "alpha", nil, true},
		{"non-nil empty list means none", "alpha", []string{}, false},
		{"all keyword", "alpha", []string{"all"}, true},
		{"ALL keyword case-insensitive", "alpha", []string{"ALL"}, true},
		{"All keyword case-insensitive", "alpha", []string{"All"}, true},
		{"matching type", "alpha", []string{"alpha"}, true},
		{"non-matching type", "alpha", []string{"beta"}, false},
		{"multiple types includes match", "beta", []string{"alpha", "beta"}, true},
		{"multiple types no match", "gamma", []string{"alpha", "beta"}, false},
		{"all mixed with specific types", "gamma", []string{"all", "alpha"}, true},
		{"duplicate types", "alpha", []string{"alpha", "alpha"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsNukeable(tt.resourceType, tt.resourceTypes))
		})
	}
}

func TestIsValidResourceType(t *testing.T) {
	t.Parallel()

	assert.True(t, IsValidResourceType("alpha", testResourceTypes))
	assert.True(t, IsValidResourceType("beta", testResourceTypes))
	assert.False(t, IsValidResourceType("nonexistent", testResourceTypes))
	assert.False(t, IsValidResourceType("", testResourceTypes))
	assert.False(t, IsValidResourceType("Alpha", testResourceTypes)) // case-sensitive
}

func TestEnsureValidResourceTypes(t *testing.T) {
	t.Parallel()

	t.Run("valid types pass", func(t *testing.T) {
		t.Parallel()
		result, err := EnsureValidResourceTypes([]string{"alpha", "beta"}, testResourceTypes)
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha", "beta"}, result)
	})

	t.Run("all keyword passes", func(t *testing.T) {
		t.Parallel()
		result, err := EnsureValidResourceTypes([]string{"all"}, testResourceTypes)
		require.NoError(t, err)
		assert.Equal(t, []string{"all"}, result)
	})

	t.Run("ALL keyword case-insensitive", func(t *testing.T) {
		t.Parallel()
		result, err := EnsureValidResourceTypes([]string{"ALL"}, testResourceTypes)
		require.NoError(t, err)
		assert.Equal(t, []string{"ALL"}, result)
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		t.Parallel()
		_, err := EnsureValidResourceTypes([]string{"nonexistent"}, testResourceTypes)
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("mixed valid and invalid returns error with all invalid listed", func(t *testing.T) {
		t.Parallel()
		_, err := EnsureValidResourceTypes([]string{"alpha", "bad1", "bad2"}, testResourceTypes)
		require.Error(t, err)
		var target InvalidResourceTypesSuppliedError
		require.ErrorAs(t, err, &target)
		assert.Equal(t, []string{"bad1", "bad2"}, target.InvalidTypes)
	})
}

func TestHandleResourceTypeSelections(t *testing.T) {
	t.Parallel()

	t.Run("empty slices returns nil (all types)", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{}, []string{}, testResourceTypes)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil slices returns nil (all types)", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections(nil, nil, testResourceTypes)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("include valid types returns those types", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{"alpha"}, []string{}, testResourceTypes)
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha"}, result)
	})

	t.Run("include all keyword", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{"all"}, []string{}, testResourceTypes)
		require.NoError(t, err)
		assert.Equal(t, []string{"all"}, result)
	})

	t.Run("include ALL keyword case-insensitive", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{"ALL"}, []string{}, testResourceTypes)
		require.NoError(t, err)
		assert.Equal(t, []string{"ALL"}, result)
	})

	t.Run("exclude valid type returns complement", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{}, []string{"gamma"}, testResourceTypes)
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha", "beta"}, result)
	})

	t.Run("exclude all types individually returns non-nil empty", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{}, []string{"alpha", "beta", "gamma"}, testResourceTypes)
		require.NoError(t, err)
		assert.NotNil(t, result, "should be non-nil empty, not nil (nil means all)")
		assert.Empty(t, result)
		// Verify IsNukeable correctly rejects all types for this result
		assert.False(t, IsNukeable("alpha", result))
		assert.False(t, IsNukeable("beta", result))
	})

	t.Run("exclude all keyword returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{}, []string{"all"}, testResourceTypes)
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("exclude ALL keyword case-insensitive returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{}, []string{"ALL"}, testResourceTypes)
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("both flags returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{"alpha"}, []string{"beta"}, testResourceTypes)
		require.Error(t, err)
		assert.IsType(t, ResourceTypeAndExcludeFlagsBothPassedError{}, err)
	})

	t.Run("invalid include type returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{"nonexistent"}, []string{}, testResourceTypes)
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("invalid exclude type returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{}, []string{"nonexistent"}, testResourceTypes)
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("duplicate include types accepted", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{"alpha", "alpha"}, []string{}, testResourceTypes)
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha", "alpha"}, result)
	})

	t.Run("no filter then IsNukeable returns true", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections(nil, nil, testResourceTypes)
		require.NoError(t, err)
		assert.True(t, IsNukeable("alpha", result))
		assert.True(t, IsNukeable("anything", result))
	})
}

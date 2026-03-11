package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNukeable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		resourceType  string
		resourceTypes []string
		expected      bool
	}{
		{"nil list means all", "gcs-bucket", nil, true},
		{"non-nil empty list means none", "gcs-bucket", []string{}, false},
		{"all keyword", "gcs-bucket", []string{"all"}, true},
		{"ALL keyword case-insensitive", "gcs-bucket", []string{"ALL"}, true},
		{"matching type", "gcs-bucket", []string{"gcs-bucket"}, true},
		{"non-matching type", "gcs-bucket", []string{"cloud-function"}, false},
		{"multiple types includes match", "cloud-function", []string{"gcs-bucket", "cloud-function"}, true},
		{"multiple types no match", "gcs-bucket", []string{"cloud-function"}, false},
		{"all mixed with specific types", "cloud-function", []string{"all", "gcs-bucket"}, true},
		{"duplicate types", "gcs-bucket", []string{"gcs-bucket", "gcs-bucket"}, true},
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

	assert.True(t, IsValidResourceType("gcs-bucket"))
	assert.True(t, IsValidResourceType("cloud-function"))
	assert.False(t, IsValidResourceType("nonexistent"))
	assert.False(t, IsValidResourceType(""))
	assert.False(t, IsValidResourceType("GCS-BUCKET")) // case-sensitive
}

func TestListResourceTypes(t *testing.T) {
	t.Parallel()

	types := ListResourceTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "gcs-bucket")
	assert.Contains(t, types, "cloud-function")

	// Verify sorted order
	for i := 1; i < len(types); i++ {
		assert.True(t, types[i-1] <= types[i], "ListResourceTypes should return sorted results")
	}
}

func TestHandleResourceTypeSelections(t *testing.T) {
	t.Parallel()

	t.Run("empty slices returns nil (all types)", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{}, []string{})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil slices returns nil (all types)", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections(nil, nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("include valid types returns those types", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{"gcs-bucket"}, []string{})
		require.NoError(t, err)
		assert.Equal(t, []string{"gcs-bucket"}, result)
	})

	t.Run("include all keyword", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{"all"}, []string{})
		require.NoError(t, err)
		assert.Equal(t, []string{"all"}, result)
	})

	t.Run("include ALL keyword case-insensitive", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{"ALL"}, []string{})
		require.NoError(t, err)
		assert.Equal(t, []string{"ALL"}, result)
	})

	t.Run("exclude valid type returns complement", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{}, []string{"cloud-function"})
		require.NoError(t, err)
		assert.Equal(t, []string{"gcs-bucket"}, result)
	})

	t.Run("exclude all types individually returns non-nil empty", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{}, []string{"gcs-bucket", "cloud-function"})
		require.NoError(t, err)
		assert.NotNil(t, result, "should be non-nil empty, not nil (nil means all)")
		assert.Empty(t, result)
		// Verify IsNukeable correctly rejects all types
		assert.False(t, IsNukeable("gcs-bucket", result))
	})

	t.Run("exclude all keyword returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{}, []string{"all"})
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("exclude ALL keyword case-insensitive returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{}, []string{"ALL"})
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("both flags returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{"gcs-bucket"}, []string{"cloud-function"})
		require.Error(t, err)
		assert.IsType(t, ResourceTypeAndExcludeFlagsBothPassedError{}, err)
	})

	t.Run("invalid include type returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{"nonexistent"}, []string{})
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("invalid exclude type returns error", func(t *testing.T) {
		t.Parallel()
		_, err := HandleResourceTypeSelections([]string{}, []string{"nonexistent"})
		require.Error(t, err)
		assert.IsType(t, InvalidResourceTypesSuppliedError{}, err)
	})

	t.Run("duplicate include types accepted", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections([]string{"gcs-bucket", "gcs-bucket"}, []string{})
		require.NoError(t, err)
		assert.Equal(t, []string{"gcs-bucket", "gcs-bucket"}, result)
	})

	t.Run("no filter then IsNukeable returns true", func(t *testing.T) {
		t.Parallel()
		result, err := HandleResourceTypeSelections(nil, nil)
		require.NoError(t, err)
		assert.True(t, IsNukeable("gcs-bucket", result))
		assert.True(t, IsNukeable("anything", result))
	})
}

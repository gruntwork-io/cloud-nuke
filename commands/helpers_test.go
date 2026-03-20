package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTagFlags(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		tags, err := parseTagFlags(nil)
		require.NoError(t, err)
		assert.Nil(t, tags)
	})

	t.Run("valid key=value with regex", func(t *testing.T) {
		tags, err := parseTagFlags([]string{"repo=^terraform-aws-.*$"})
		require.NoError(t, err)
		expr := tags["repo"]
		assert.True(t, expr.RE.MatchString("terraform-aws-data-storage"))
		assert.False(t, expr.RE.MatchString("something-else"))
	})

	t.Run("value containing equals sign", func(t *testing.T) {
		tags, err := parseTagFlags([]string{"key=val=ue"})
		require.NoError(t, err)
		expr := tags["key"]
		assert.True(t, expr.RE.MatchString("val=ue"))
	})

	t.Run("multiple tags", func(t *testing.T) {
		tags, err := parseTagFlags([]string{"repo=data-storage", "env=ci"})
		require.NoError(t, err)
		require.Len(t, tags, 2)
	})

	t.Run("whitespace is trimmed", func(t *testing.T) {
		tags, err := parseTagFlags([]string{" repo = data-storage "})
		require.NoError(t, err)
		require.Contains(t, tags, "repo")
	})

	t.Run("format errors", func(t *testing.T) {
		for _, input := range []string{"noequals", "=value", "key="} {
			t.Run(input, func(t *testing.T) {
				_, err := parseTagFlags([]string{input})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "Invalid tag format")
			})
		}
	})

	t.Run("invalid regex error", func(t *testing.T) {
		_, err := parseTagFlags([]string{"key=[invalid(regex"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid regex")
	})

	t.Run("duplicate key returns error", func(t *testing.T) {
		_, err := parseTagFlags([]string{"env=project1", "env=project2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Duplicate tag key")
		assert.Contains(t, err.Error(), "env")
	})

	t.Run("partial regex match", func(t *testing.T) {
		tags, err := parseTagFlags([]string{"env=dev"})
		require.NoError(t, err)
		expr := tags["env"]
		// Regex is a partial match by default - "dev" matches "dev-staging"
		assert.True(t, expr.RE.MatchString("dev"), "should match exact value")
		assert.True(t, expr.RE.MatchString("dev-staging"), "should match partial (substring)")
		assert.True(t, expr.RE.MatchString("my-dev-env"), "should match partial (middle)")
		assert.False(t, expr.RE.MatchString("production"), "should not match unrelated value")
	})

	t.Run("exact match with anchors", func(t *testing.T) {
		tags, err := parseTagFlags([]string{"env=^dev$"})
		require.NoError(t, err)
		expr := tags["env"]
		assert.True(t, expr.RE.MatchString("dev"), "should match exact value")
		assert.False(t, expr.RE.MatchString("dev-staging"), "anchored regex should not match partial")
	})
}

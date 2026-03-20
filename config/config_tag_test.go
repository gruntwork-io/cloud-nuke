package config

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldIncludeWithTagFilters(t *testing.T) {
	withInclude := ResourceType{
		IncludeRule: FilterRule{
			Tags: map[string]Expression{
				"env": {RE: *regexp.MustCompile("dev")},
			},
		},
	}

	withExclude := ResourceType{
		ExcludeRule: FilterRule{
			Tags: map[string]Expression{
				"env": {RE: *regexp.MustCompile("prod")},
			},
		},
	}

	withBoth := ResourceType{
		IncludeRule: FilterRule{
			Tags: map[string]Expression{
				"env": {RE: *regexp.MustCompile("dev")},
			},
		},
		ExcludeRule: FilterRule{
			Tags: map[string]Expression{
				"team": {RE: *regexp.MustCompile("finance")},
			},
		},
	}

	tests := []struct {
		name         string
		resourceType ResourceType
		resourceTags map[string]string
		expected     bool
	}{
		{"matching include tag", withInclude, map[string]string{"env": "dev"}, true},
		{"non-matching include tag", withInclude, map[string]string{"env": "prod"}, false},
		{"nil tags with include rule", withInclude, nil, false},
		{"empty tags with include rule", withInclude, map[string]string{}, false},
		{"matching exclude tag", withExclude, map[string]string{"env": "prod"}, false},
		{"non-matching exclude tag", withExclude, map[string]string{"env": "dev"}, true},
		{"nil tags with exclude rule only", withExclude, nil, true},
		{"include match + exclude match", withBoth, map[string]string{"env": "dev", "team": "finance"}, false},
		{"include match + exclude miss", withBoth, map[string]string{"env": "dev", "team": "eng"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.resourceType.ShouldInclude(ResourceValue{Tags: tc.resourceTags})
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAddIncludeTags(t *testing.T) {
	t.Run("applies to all resource types with AND logic", func(t *testing.T) {
		cfg := &Config{}
		tags := map[string]Expression{
			"gruntwork-repo": {RE: *regexp.MustCompile("^terraform-aws-data-storage$")},
		}
		cfg.AddIncludeTags(tags)

		for _, rt := range cfg.allResourceTypes() {
			require.Contains(t, rt.IncludeRule.Tags, "gruntwork-repo")
			assert.Equal(t, "AND", rt.IncludeRule.TagsOperator)
		}
	})

	t.Run("config file tags take precedence", func(t *testing.T) {
		cfg := &Config{}
		cfg.EC2.IncludeRule.Tags = map[string]Expression{
			"gruntwork-repo": {RE: *regexp.MustCompile("^terraform-aws-eks$")},
		}

		cfg.AddIncludeTags(map[string]Expression{
			"gruntwork-repo": {RE: *regexp.MustCompile("^terraform-aws-data-storage$")},
		})

		// EC2 keeps the config file value, not the CLI value
		re := cfg.EC2.IncludeRule.Tags["gruntwork-repo"].RE
		assert.True(t, re.MatchString("terraform-aws-eks"))
		assert.False(t, re.MatchString("terraform-aws-data-storage"))
	})

	t.Run("config file operator is preserved", func(t *testing.T) {
		cfg := &Config{}
		cfg.EC2.IncludeRule.TagsOperator = "OR"

		cfg.AddIncludeTags(map[string]Expression{
			"repo": {RE: *regexp.MustCompile("foo")},
		})

		assert.Equal(t, "OR", cfg.EC2.IncludeRule.TagsOperator)
	})

	t.Run("merges different tag keys from CLI and config", func(t *testing.T) {
		cfg := &Config{}
		cfg.EC2.IncludeRule.Tags = map[string]Expression{
			"env": {RE: *regexp.MustCompile("^prod$")},
		}

		cfg.AddIncludeTags(map[string]Expression{
			"gruntwork-repo": {RE: *regexp.MustCompile("^terraform-aws-data-storage$")},
		})

		require.Len(t, cfg.EC2.IncludeRule.Tags, 2)
		assert.Contains(t, cfg.EC2.IncludeRule.Tags, "env")
		assert.Contains(t, cfg.EC2.IncludeRule.Tags, "gruntwork-repo")
	})

	t.Run("nil and empty are no-ops", func(t *testing.T) {
		cfg := &Config{}
		cfg.AddIncludeTags(nil)
		cfg.AddIncludeTags(map[string]Expression{})

		for _, rt := range cfg.allResourceTypes() {
			assert.Nil(t, rt.IncludeRule.Tags)
		}
	})

	t.Run("multiple tags require all to match", func(t *testing.T) {
		cfg := &Config{}
		cfg.AddIncludeTags(map[string]Expression{
			"gruntwork-repo": {RE: *regexp.MustCompile("^data-storage$")},
			"environment":    {RE: *regexp.MustCompile("^ci$")},
		})

		rt := cfg.allResourceTypes()[0]

		// Both match → include
		assert.True(t, rt.ShouldIncludeBasedOnTag(map[string]string{
			"gruntwork-repo": "data-storage",
			"environment":    "ci",
		}))
		// One mismatch → exclude
		assert.False(t, rt.ShouldIncludeBasedOnTag(map[string]string{
			"gruntwork-repo": "data-storage",
			"environment":    "prod",
		}))
		// Missing tag → exclude
		assert.False(t, rt.ShouldIncludeBasedOnTag(map[string]string{
			"gruntwork-repo": "data-storage",
		}))
		// Nil tags → exclude
		assert.False(t, rt.ShouldIncludeBasedOnTag(nil))
	})
}

func TestIncludeTagsWithTimeFilters(t *testing.T) {
	cfg := &Config{}
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	cfg.AddExcludeAfterTime(&twoHoursAgo)
	cfg.AddIncludeTags(map[string]Expression{
		"gruntwork-repo": {RE: *regexp.MustCompile("^data-storage$")},
	})

	rt := cfg.allResourceTypes()[0]
	oldTime := time.Now().Add(-3 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	goodTags := map[string]string{"gruntwork-repo": "data-storage"}
	badTags := map[string]string{"gruntwork-repo": "eks"}

	// Old + matching tag → include
	assert.True(t, rt.ShouldInclude(ResourceValue{Time: &oldTime, Tags: goodTags}))
	// New + matching tag → exclude (too new)
	assert.False(t, rt.ShouldInclude(ResourceValue{Time: &newTime, Tags: goodTags}))
	// Old + wrong tag → exclude
	assert.False(t, rt.ShouldInclude(ResourceValue{Time: &oldTime, Tags: badTags}))
	// Old + nil tags → exclude
	assert.False(t, rt.ShouldInclude(ResourceValue{Time: &oldTime, Tags: nil}))
}

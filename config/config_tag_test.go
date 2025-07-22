package config

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldIncludeWithTagFilters(t *testing.T) {
	// Create a resource type with tag inclusion rules
	resourceTypeWithIncludeTags := ResourceType{
		IncludeRule: FilterRule{
			Tags: map[string]Expression{
				"env": {RE: *regexp.MustCompile("dev")},
			},
		},
	}

	// Create a resource type with tag exclusion rules
	resourceTypeWithExcludeTags := ResourceType{
		ExcludeRule: FilterRule{
			Tags: map[string]Expression{
				"env": {RE: *regexp.MustCompile("prod")},
			},
		},
	}

	// Create a resource type with both inclusion and exclusion rules
	resourceTypeWithBothRules := ResourceType{
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
		{
			name:         "Resource with matching include tag should be included",
			resourceType: resourceTypeWithIncludeTags,
			resourceTags: map[string]string{"env": "dev", "team": "engineering"},
			expected:     true,
		},
		{
			name:         "Resource with non-matching include tag should be excluded",
			resourceType: resourceTypeWithIncludeTags,
			resourceTags: map[string]string{"env": "prod", "team": "engineering"},
			expected:     false,
		},
		{
			name:         "Resource with no tags should be excluded when include tag rule exists",
			resourceType: resourceTypeWithIncludeTags,
			resourceTags: nil,
			expected:     false,
		},
		{
			name:         "Resource with empty tags should be excluded when include tag rule exists",
			resourceType: resourceTypeWithIncludeTags,
			resourceTags: map[string]string{},
			expected:     false,
		},
		{
			name:         "Resource with matching exclude tag should be excluded",
			resourceType: resourceTypeWithExcludeTags,
			resourceTags: map[string]string{"env": "prod", "team": "engineering"},
			expected:     false,
		},
		{
			name:         "Resource with non-matching exclude tag should be included",
			resourceType: resourceTypeWithExcludeTags,
			resourceTags: map[string]string{"env": "dev", "team": "engineering"},
			expected:     true,
		},
		{
			name:         "Resource with no tags should be included when only exclude tag rule exists",
			resourceType: resourceTypeWithExcludeTags,
			resourceTags: nil,
			expected:     true,
		},
		{
			name:         "Resource with matching include tag but also matching exclude tag should be excluded",
			resourceType: resourceTypeWithBothRules,
			resourceTags: map[string]string{"env": "dev", "team": "finance"},
			expected:     false,
		},
		{
			name:         "Resource with matching include tag and non-matching exclude tag should be included",
			resourceType: resourceTypeWithBothRules,
			resourceTags: map[string]string{"env": "dev", "team": "engineering"},
			expected:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resourceValue := ResourceValue{
				Tags: tc.resourceTags,
			}
			result := tc.resourceType.ShouldInclude(resourceValue)
			assert.Equal(t, tc.expected, result)
		})
	}
}

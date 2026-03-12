package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllRegisteredResources_NonEmpty(t *testing.T) {
	resources := GetAllRegisteredResources()
	require.NotEmpty(t, resources, "registered resources should not be empty")
}

func TestGetAllRegisteredResources_AllHaveResourceName(t *testing.T) {
	resources := GetAllRegisteredResources()
	for _, r := range resources {
		name := (*r).ResourceName()
		assert.NotEmpty(t, name, "every registered resource must have a non-empty ResourceName()")
	}
}

func TestGetAllRegisteredResources_NoDuplicateNames(t *testing.T) {
	resources := GetAllRegisteredResources()
	seen := make(map[string]bool)
	for _, r := range resources {
		name := (*r).ResourceName()
		assert.False(t, seen[name], "duplicate resource name found: %s", name)
		seen[name] = true
	}
}

func TestRegisteredResources_GlobalAndRegionalCounts(t *testing.T) {
	globalResources := getRegisteredGlobalResources()
	regionalResources := getRegisteredRegionalResources()
	allResources := GetAllRegisteredResources()

	assert.Greater(t, len(globalResources), 0, "should have global resources")
	assert.Greater(t, len(regionalResources), 0, "should have regional resources")
	assert.Equal(t, len(globalResources)+len(regionalResources), len(allResources),
		"total resources should equal global + regional")
}

func TestRegisteredResources_GlobalContainsExpectedResources(t *testing.T) {
	globalResources := getRegisteredGlobalResources()
	names := make([]string, len(globalResources))
	for i, r := range globalResources {
		names[i] = r.ResourceName()
	}

	// IAM resources should be in global list
	assert.Contains(t, names, "iam-user")
	assert.Contains(t, names, "iam-role")
	assert.Contains(t, names, "iam-policy")
	// S3 should be in global list
	assert.Contains(t, names, "s3")
}

func TestRegisteredResources_RegionalContainsExpectedResources(t *testing.T) {
	regionalResources := getRegisteredRegionalResources()
	names := make([]string, len(regionalResources))
	for i, r := range regionalResources {
		names[i] = r.ResourceName()
	}

	// Spot-check some common regional resources
	assert.Contains(t, names, "ec2")
	assert.Contains(t, names, "lambda")
	assert.Contains(t, names, "vpc")
}

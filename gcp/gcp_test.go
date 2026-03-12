package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNukeable_EmptyLists(t *testing.T) {
	// Empty include and exclude lists means everything is nukeable
	assert.True(t, IsNukeable("SomeResource", nil, nil))
	assert.True(t, IsNukeable("AnyResource", []string{}, []string{}))
}

func TestIsNukeable_AllKeyword(t *testing.T) {
	assert.True(t, IsNukeable("SomeResource", []string{"all"}, nil))
	assert.True(t, IsNukeable("AnotherResource", []string{"all"}, nil))
}

func TestIsNukeable_SpecificIncludeList(t *testing.T) {
	includeList := []string{"ResourceA", "ResourceB"}

	assert.True(t, IsNukeable("ResourceA", includeList, nil))
	assert.True(t, IsNukeable("ResourceB", includeList, nil))
	assert.False(t, IsNukeable("ResourceC", includeList, nil))
}

func TestIsNukeable_ExcludeList(t *testing.T) {
	excludeList := []string{"ResourceX"}

	// Excluded resource is not nukeable even with empty include list
	assert.False(t, IsNukeable("ResourceX", nil, excludeList))
	assert.False(t, IsNukeable("ResourceX", []string{}, excludeList))

	// Non-excluded resources are still nukeable
	assert.True(t, IsNukeable("ResourceY", nil, excludeList))
}

func TestIsNukeable_ExcludeOverridesInclude(t *testing.T) {
	includeList := []string{"ResourceA", "ResourceB"}
	excludeList := []string{"ResourceA"}

	// ResourceA is in both include and exclude, exclude wins
	assert.False(t, IsNukeable("ResourceA", includeList, excludeList))
	assert.True(t, IsNukeable("ResourceB", includeList, excludeList))
}

func TestIsNukeable_ExcludeOverridesAll(t *testing.T) {
	// "all" includes everything, but exclude still takes precedence
	assert.False(t, IsNukeable("Excluded", []string{"all"}, []string{"Excluded"}))
	assert.True(t, IsNukeable("Included", []string{"all"}, []string{"Excluded"}))
}

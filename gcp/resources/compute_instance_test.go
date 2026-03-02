package resources

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
)

func TestComputeInstances_ResourceName(t *testing.T) {
	r := NewComputeInstances()
	assert.Equal(t, "compute-instance", r.ResourceName())
}

func TestComputeInstances_MaxBatchSize(t *testing.T) {
	r := NewComputeInstances()
	assert.Equal(t, ComputeInstanceBatchSize, r.MaxBatchSize())
}

func TestComputeInstances_ConfigGetter(t *testing.T) {
	r := NewComputeInstances()
	cfg := config.Config{
		ComputeInstance: config.ResourceType{
			Timeout: "5m",
		},
	}
	rt := r.GetAndSetResourceConfig(cfg)
	assert.Equal(t, "5m", rt.Timeout)
}

func TestParseComputeInstanceID(t *testing.T) {
	project, zone, name, err := parseComputeInstanceID("my-project/us-central1-a/my-vm")
	require.NoError(t, err)
	assert.Equal(t, "my-project", project)
	assert.Equal(t, "us-central1-a", zone)
	assert.Equal(t, "my-vm", name)
}

func TestParseComputeInstanceID_SlashesInName(t *testing.T) {
	// SplitN with limit 3 captures everything after second "/" as the name
	project, zone, name, err := parseComputeInstanceID("project/zone/name/with/slashes")
	require.NoError(t, err)
	assert.Equal(t, "project", project)
	assert.Equal(t, "zone", zone)
	assert.Equal(t, "name/with/slashes", name)
}

func TestParseComputeInstanceID_Invalid(t *testing.T) {
	tests := []string{
		"",
		"only-one",
		"two/parts",
		"//empty-parts",
		"/zone/name",
		"project//name",
		"project/zone/",
	}
	for _, id := range tests {
		_, _, _, err := parseComputeInstanceID(id)
		assert.Error(t, err, "expected error for ID: %q", id)
	}
}

func TestExtractZoneName(t *testing.T) {
	assert.Equal(t, "us-central1-a", extractZoneName("zones/us-central1-a"))
	assert.Equal(t, "europe-west1-b", extractZoneName("zones/europe-west1-b"))
	assert.Equal(t, "already-bare", extractZoneName("already-bare"))
}

func TestIsGCPNotFound(t *testing.T) {
	t.Run("404 error returns true", func(t *testing.T) {
		err := &googleapi.Error{Code: 404, Message: "not found"}
		assert.True(t, isGCPNotFound(err))
	})
	t.Run("403 error returns false", func(t *testing.T) {
		err := &googleapi.Error{Code: 403, Message: "forbidden"}
		assert.False(t, isGCPNotFound(err))
	})
	t.Run("non-googleapi error returns false", func(t *testing.T) {
		err := assert.AnError
		assert.False(t, isGCPNotFound(err))
	})
	t.Run("wrapped 404 error returns true", func(t *testing.T) {
		inner := &googleapi.Error{Code: 404, Message: "not found"}
		err := fmt.Errorf("outer context: %w", inner)
		assert.True(t, isGCPNotFound(err))
	})
	t.Run("nil error returns false", func(t *testing.T) {
		assert.False(t, isGCPNotFound(nil))
	})
}

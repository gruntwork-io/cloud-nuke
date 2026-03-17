package resources

import (
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeInstances_ResourceName(t *testing.T) {
	t.Parallel()
	r := NewComputeInstances()
	assert.Equal(t, "compute-instance", r.ResourceName())
}

func TestComputeInstances_MaxBatchSize(t *testing.T) {
	t.Parallel()
	r := NewComputeInstances()
	assert.Equal(t, ComputeInstanceBatchSize, r.MaxBatchSize())
}

func TestComputeInstances_ConfigGetter(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	project, zone, name, err := parseComputeInstanceID("my-project/us-central1-a/my-vm")
	require.NoError(t, err)
	assert.Equal(t, "my-project", project)
	assert.Equal(t, "us-central1-a", zone)
	assert.Equal(t, "my-vm", name)
}

func TestParseComputeInstanceID_SlashesInName(t *testing.T) {
	t.Parallel()
	// SplitN with limit 3 captures everything after second "/" as the name
	project, zone, name, err := parseComputeInstanceID("project/zone/name/with/slashes")
	require.NoError(t, err)
	assert.Equal(t, "project", project)
	assert.Equal(t, "zone", zone)
	assert.Equal(t, "name/with/slashes", name)
}

func TestParseComputeInstanceID_Invalid(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	assert.Equal(t, "us-central1-a", extractZoneName("zones/us-central1-a"))
	assert.Equal(t, "europe-west1-b", extractZoneName("zones/europe-west1-b"))
	assert.Equal(t, "already-bare", extractZoneName("already-bare"))
}

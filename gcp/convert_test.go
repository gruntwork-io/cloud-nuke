package gcp

import (
	"testing"

	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFoundRoundTripsThroughTheProjectShape pins that the value this package
// has always returned survives the trip to the engine and back. Dropping a
// resource in either direction is a nuke that deletes nothing and reports
// success.
func TestFoundRoundTripsThroughTheProjectShape(t *testing.T) {
	t.Parallel()

	registered := GetAndInitRegisteredResources(resources.GcpConfig{ProjectID: "my-project"})
	require.NotEmpty(t, registered)

	project := &GcpProjectResources{Resources: map[string]GcpResources{
		"my-project": {Resources: registered[:2]},
	}}

	found := foundFrom(project, "my-project")
	require.Len(t, found.Targets, 1)
	assert.Equal(t, "my-project", found.Targets[0].ID)
	require.Len(t, found.Resources["my-project"], 2)

	back, err := projectResourcesFrom(found)
	require.NoError(t, err)
	require.Len(t, back.Resources["my-project"].Resources, 2)
	assert.Equal(t,
		(*registered[0]).ResourceName(),
		(*back.Resources["my-project"].Resources[0]).ResourceName())
}

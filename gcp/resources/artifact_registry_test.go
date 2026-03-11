package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArtifactRegistryRepositories_ResourceName(t *testing.T) {
	ar := NewArtifactRegistryRepositories()
	assert.Equal(t, "artifact-registry", ar.ResourceName())
}

func TestArtifactRegistryRepositories_MaxBatchSize(t *testing.T) {
	ar := NewArtifactRegistryRepositories()
	assert.Equal(t, 50, ar.MaxBatchSize())
}

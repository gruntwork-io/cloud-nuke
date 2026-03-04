package resources

import (
	"context"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGcpResourceAdapter_InitPanicRecovery(t *testing.T) {
	r := NewGcpResource(&resource.Resource[*string]{
		ResourceTypeName: "test-resource",
		BatchSize:        10,
		InitClient: func(r *resource.Resource[*string], cfg any) {
			panic("simulated credential failure")
		},
		Lister: func(ctx context.Context, client *string, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			t.Fatal("lister should not be called after init failure")
			return nil, nil
		},
	})

	// Init should not panic
	require.NotPanics(t, func() {
		r.Init("test-project")
	})

	// GetAndSetIdentifiers should return the init error
	_, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initialization failed")
	assert.Contains(t, err.Error(), "simulated credential failure")

	// Nuke should also return the init error
	_, err = r.Nuke(context.Background(), []string{"some-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initialization failed")
}

func TestGcpResourceAdapter_InitSuccess(t *testing.T) {
	var initCalled bool
	r := NewGcpResource(&resource.Resource[*string]{
		ResourceTypeName: "test-resource",
		BatchSize:        10,
		InitClient: func(r *resource.Resource[*string], cfg any) {
			initCalled = true
		},
	})

	r.Init("test-project")
	assert.True(t, initCalled)
}

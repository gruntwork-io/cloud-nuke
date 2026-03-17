package resources

import (
	"context"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGcpResourceAdapter_InitRecoversPanic(t *testing.T) {
	t.Parallel()
	adapter := NewGcpResource(&resource.Resource[any]{
		ResourceTypeName: "test-panic-resource",
		InitClient: func(r *resource.Resource[any], cfg any) {
			panic("simulated client init failure")
		},
	})

	// Init should not panic
	assert.NotPanics(t, func() {
		adapter.Init(GcpConfig{ProjectID: "test-project"})
	})
}

func TestGcpResourceAdapter_GetAndSetIdentifiersReturnsInitErr(t *testing.T) {
	t.Parallel()
	adapter := NewGcpResource(&resource.Resource[any]{
		ResourceTypeName: "test-panic-identifiers",
		InitClient: func(r *resource.Resource[any], cfg any) {
			panic("simulated client init failure")
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return config.ResourceType{}
		},
		Lister: func(ctx context.Context, client any, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return nil, nil
		},
	})

	adapter.Init(GcpConfig{ProjectID: "test-project"})

	ids, err := adapter.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "panic during Init")
	assert.Contains(t, err.Error(), "simulated client init failure")
}

func TestGcpResourceAdapter_NoPanicWorksNormally(t *testing.T) {
	t.Parallel()
	adapter := NewGcpResource(&resource.Resource[any]{
		ResourceTypeName: "test-normal-resource",
		InitClient: func(r *resource.Resource[any], cfg any) {
			// No panic - normal initialization
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return config.ResourceType{}
		},
		Lister: func(ctx context.Context, client any, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return nil, nil
		},
	})

	adapter.Init(GcpConfig{ProjectID: "test-project"})

	ids, err := adapter.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.NoError(t, err)
	assert.Empty(t, ids)
}

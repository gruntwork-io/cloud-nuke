package resources

import (
	"context"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitReportsAClientFailureRatherThanPanicking pins that a resource whose
// client cannot be built reports itself and lets the rest of the run continue.
// Three resources used to panic here, under a comment claiming this adapter
// recovered it, and nothing did: the panic is raised inside the scan's errgroup
// goroutine where the CLI's panic handler cannot reach it, so cloud-nuke died
// with a stack trace when a credential was missing.
func TestInitReportsAClientFailureRatherThanPanicking(t *testing.T) {
	t.Parallel()

	r := NewGcpResource(&resource.Resource[any]{
		ResourceTypeName: "fake-init-panic-on-list",
		InitClient: WrapGcpInitClient(func(r *resource.Resource[any], cfg GcpConfig) {
			panic("failed to create client: no credentials")
		}),
		ConfigGetter: func(c config.Config) config.ResourceType { return config.ResourceType{} },
		Lister: func(ctx context.Context, client any, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			t.Fatal("the lister must not run when the client could not be built")
			return nil, nil
		},
	})

	// Init returns rather than taking the process down.
	r.Init(GcpConfig{ProjectID: "p"})

	// The error reaches the caller through the list, which is where the engine
	// turns it into one row in the errors table.
	_, err := r.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fake-init-panic-on-list")
	assert.Contains(t, err.Error(), "no credentials")
}

// TestNukeReportsAClientFailureRatherThanDeletingNothingQuietly pins the same
// for the delete path, so a resource whose client is missing is not silently
// treated as having nothing to delete.
func TestNukeReportsAClientFailureRatherThanDeletingNothingQuietly(t *testing.T) {
	t.Parallel()

	r := NewGcpResource(&resource.Resource[any]{
		ResourceTypeName: "fake-init-panic-on-nuke",
		InitClient: WrapGcpInitClient(func(r *resource.Resource[any], cfg GcpConfig) {
			panic("failed to create client: no credentials")
		}),
		ConfigGetter: func(c config.Config) config.ResourceType { return config.ResourceType{} },
	})
	r.Init(GcpConfig{ProjectID: "p"})

	_, err := r.Nuke(context.Background(), []string{"something"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no credentials")
}

package resource

import (
	"context"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
)

// CloudResource represents a cloud resource that can be discovered and deleted
// across any cloud provider (AWS, GCP, Azure, etc.)
type CloudResource interface {
	// Resource identification
	ResourceName() string
	ResourceIdentifiers() []string

	// Batching control
	MaxBatchSize() int

	// Resource operations
	GetAndSetIdentifiers(context.Context, config.Config) ([]string, error)
	Nuke(identifiers []string) error
	IsNukable(identifier string) (bool, error)

	// Configuration
	GetAndSetResourceConfig(config.Config) config.ResourceType
	PrepareContext(context.Context, config.ResourceType) error
}

// BaseCloudResource provides common functionality for all cloud resources
// Both AWS and GCP base resources share these exact fields
type BaseCloudResource struct {
	Nukables map[string]error // Tracks nukable status per identifier
	Timeout  time.Duration
	Context  context.Context
	cancel   context.CancelFunc
}

// GetNukableStatus retrieves the nukable status for a given identifier
func (b *BaseCloudResource) GetNukableStatus(identifier string) (error, bool) {
	val, ok := b.Nukables[identifier]
	return val, ok
}

// SetNukableStatus sets the nukable status for a given identifier
func (b *BaseCloudResource) SetNukableStatus(identifier string, err error) {
	if b.Nukables == nil {
		b.Nukables = make(map[string]error)
	}
	b.Nukables[identifier] = err
}

// IsNukable checks if an identifier is nukable based on its stored status
func (b *BaseCloudResource) IsNukable(identifier string) (bool, error) {
	if err, ok := b.GetNukableStatus(identifier); ok {
		return err == nil, err
	}
	return true, nil
}

// PrepareContext sets up the context with timeout if specified in config
func (b *BaseCloudResource) PrepareContext(parentContext context.Context, resourceConfig config.ResourceType) error {
	if resourceConfig.Timeout == "" {
		b.Context = parentContext
		return nil
	}

	duration, err := time.ParseDuration(resourceConfig.Timeout)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parentContext, duration)
	b.Context = ctx
	b.cancel = cancel
	b.Timeout = duration

	return nil
}

// CancelContext cancels the context if it was created with a timeout
func (b *BaseCloudResource) CancelContext() {
	if b.cancel != nil {
		b.cancel()
	}
}

// HasCancelFunc returns true if a cancel function exists (for testing)
func (b *BaseCloudResource) HasCancelFunc() bool {
	return b.cancel != nil
}

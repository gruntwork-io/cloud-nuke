package resources

import (
	"context"
	"errors"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
)

const maxRetries = 3
const waitDuration = 5 * time.Second

// BaseGcpResource struct and its associated methods to serve as a placeholder or template for a resource that is not
// yet fully implemented within a system or framework. Its purpose is to provide a skeleton structure that adheres to a
// specific interface or contract expected by the system without containing the actual implementation details.
type BaseGcpResource struct {
	// A key-value of identifiers and nukable status
	Nukables map[string]error
	Timeout  time.Duration
	Context  context.Context
	cancel   context.CancelFunc
	// The GCP project ID
	ProjectID string
}

func (br *BaseGcpResource) Init(projectID string) {
	br.Nukables = make(map[string]error)
	br.ProjectID = projectID
}

func (br *BaseGcpResource) ResourceName() string {
	return "not implemented: ResourceName"
}

func (br *BaseGcpResource) ResourceIdentifiers() []string {
	return nil
}

func (br *BaseGcpResource) MaxBatchSize() int {
	return 0
}

func (br *BaseGcpResource) Nuke(_ []string) error {
	return errors.New("not implemented: Nuke")
}

func (br *BaseGcpResource) GetAndSetIdentifiers(_ context.Context, _ config.Config) ([]string, error) {
	return nil, errors.New("not implemented: GetAndSetIdentifiers")
}

func (br *BaseGcpResource) GetNukableStatus(identifier string) (error, bool) {
	val, ok := br.Nukables[identifier]
	return val, ok
}

func (br *BaseGcpResource) SetNukableStatus(identifier string, err error) {
	br.Nukables[identifier] = err
}

func (br *BaseGcpResource) IsNukable(identifier string) (bool, error) {
	if err, ok := br.GetNukableStatus(identifier); ok {
		return err == nil, err
	}
	return true, nil
}

func (br *BaseGcpResource) GetAndSetResourceConfig(_ config.Config) config.ResourceType {
	return config.ResourceType{
		Timeout: "",
	}
}

// PrepareContext creates a new context with timeout for the resource operation
func (br *BaseGcpResource) PrepareContext(parentContext context.Context, resourceConfig config.ResourceType) error {
	if resourceConfig.Timeout == "" {
		br.Context = parentContext
		return nil
	}

	duration, err := time.ParseDuration(resourceConfig.Timeout)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parentContext, duration)
	br.Context = ctx
	br.cancel = cancel
	return nil
}

// CancelContext cancels the context for the resource operation
func (br *BaseGcpResource) CancelContext() {
	if br.cancel != nil {
		br.cancel()
	}
}

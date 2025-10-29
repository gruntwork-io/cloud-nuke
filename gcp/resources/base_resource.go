package resources

import (
	"context"
	"errors"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

const maxRetries = 3
const waitDuration = 5 * time.Second

// BaseGcpResource embeds BaseCloudResource and adds GCP-specific functionality
// It serves as a base for all GCP resource implementations
type BaseGcpResource struct {
	resource.BaseCloudResource // Embedded cloud-agnostic base
	// The GCP project ID (GCP-specific field)
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

func (br *BaseGcpResource) GetAndSetResourceConfig(_ config.Config) config.ResourceType {
	return config.ResourceType{
		Timeout: "",
	}
}

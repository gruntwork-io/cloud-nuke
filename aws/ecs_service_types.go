package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// ECSServices - Represents all ECS services found in a region
type ECSServices struct {
	Services          []string
	ServiceClusterMap map[string]string
}

// ResourceName - The simple name of the aws resource
func (services ECSServices) ResourceName() string {
	return "ecsserv"
}

// ResourceIdentifiers - The ARNs of the collected ECS services
func (services ECSServices) ResourceIdentifiers() []string {
	return services.Services
}

func (services ECSServices) MaxBatchSize() int {
	return 200
}

// Nuke - nuke all ECS service resources
func (services ECSServices) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeAllEcsServices(awsSession, services.ServiceClusterMap, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

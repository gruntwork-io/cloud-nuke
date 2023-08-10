package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// ECSServices - Represents all ECS services found in a region
type ECSServices struct {
	Client            ecsiface.ECSAPI
	Region            string
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
	return 49
}

// Nuke - nuke all ECS service resources
func (services ECSServices) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := services.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

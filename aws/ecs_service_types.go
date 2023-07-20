package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// ECSService - Represents all ECS services found in a region
type ECSService struct {
	Client            ecsiface.ECSAPI
	Region            string
	Services          []string
	ServiceClusterMap map[string]string
}

// ResourceName - The simple name of the aws resource
func (services ECSService) ResourceName() string {
	return "ecs-service"
}

// ResourceIdentifiers - The ARNs of the collected ECS services
func (services ECSService) ResourceIdentifiers() []string {
	return services.Services
}

func (services ECSService) MaxBatchSize() int {
	return 49
}

// Nuke - nuke all ECS service resources
func (services ECSService) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeAllEcsServices(awsSession, services.ServiceClusterMap, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

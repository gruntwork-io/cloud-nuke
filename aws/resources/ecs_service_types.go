package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// ECSServices - Represents all ECS services found in a region
type ECSServices struct {
	BaseAwsResource
	Client            ecsiface.ECSAPI
	Region            string
	Services          []string
	ServiceClusterMap map[string]string
}

func (services *ECSServices) Init(session *session.Session) {
	services.Client = ecs.New(session)
}

// ResourceName - The simple name of the aws resource
func (services *ECSServices) ResourceName() string {
	return "ecsserv"
}

// ResourceIdentifiers - The ARNs of the collected ECS services
func (services *ECSServices) ResourceIdentifiers() []string {
	return services.Services
}

func (services *ECSServices) MaxBatchSize() int {
	return 49
}

func (services *ECSServices) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := services.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	services.Services = awsgo.StringValueSlice(identifiers)
	return services.Services, nil
}

// Nuke - nuke all ECS service resources
func (services *ECSServices) Nuke(identifiers []string) error {
	if err := services.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

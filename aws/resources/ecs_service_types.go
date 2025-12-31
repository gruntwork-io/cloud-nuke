package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ECSServicesAPI interface {
	ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListServices(ctx context.Context, params *ecs.ListServicesInput, optFns ...func(*ecs.Options)) (*ecs.ListServicesOutput, error)
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	DeleteService(ctx context.Context, params *ecs.DeleteServiceInput, optFns ...func(*ecs.Options)) (*ecs.DeleteServiceOutput, error)
	UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
}

// ECSServices - Represents all ECS services found in a region
type ECSServices struct {
	BaseAwsResource
	Client            ECSServicesAPI
	Region            string
	Services          []string
	ServiceClusterMap map[string]string
}

func (services *ECSServices) Init(cfg aws.Config) {
	services.Client = ecs.NewFromConfig(cfg)
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

func (services *ECSServices) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ECSService
}

func (services *ECSServices) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := services.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	services.Services = aws.ToStringSlice(identifiers)
	return services.Services, nil
}

// Nuke - nuke all ECS service resources
func (services *ECSServices) Nuke(ctx context.Context, identifiers []string) error {
	if err := services.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

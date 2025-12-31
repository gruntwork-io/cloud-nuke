package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type AppRunnerServiceAPI interface {
	DeleteService(ctx context.Context, params *apprunner.DeleteServiceInput, optFns ...func(*apprunner.Options)) (*apprunner.DeleteServiceOutput, error)
	ListServices(ctx context.Context, params *apprunner.ListServicesInput, optFns ...func(*apprunner.Options)) (*apprunner.ListServicesOutput, error)
}

type AppRunnerService struct {
	BaseAwsResource
	Client     AppRunnerServiceAPI
	Region     string
	AppRunners []string
}

func (a *AppRunnerService) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.AppRunnerService
}

func (a *AppRunnerService) Init(cfg aws.Config) {
	a.Client = apprunner.NewFromConfig(cfg)
}

func (a *AppRunnerService) ResourceName() string { return "app-runner-service" }

func (a *AppRunnerService) ResourceIdentifiers() []string { return a.AppRunners }

func (a *AppRunnerService) MaxBatchSize() int { return 19 }

func (a *AppRunnerService) Nuke(ctx context.Context, identifiers []string) error {
	if err := a.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (a *AppRunnerService) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := a.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	a.AppRunners = aws.ToStringSlice(identifiers)
	return a.AppRunners, nil
}

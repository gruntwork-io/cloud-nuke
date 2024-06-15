package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apprunner"
	"github.com/aws/aws-sdk-go/service/apprunner/apprunneriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type AppRunnerService struct {
	BaseAwsResource
	Client     apprunneriface.AppRunnerAPI
	Region     string
	AppRunners []string
}

func (a *AppRunnerService) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.AppRunnerService
}

func (a *AppRunnerService) Init(session *session.Session) {
	a.Client = apprunner.New(session)
}

func (a *AppRunnerService) ResourceName() string { return "app-runner-service" }

func (a *AppRunnerService) ResourceIdentifiers() []string { return a.AppRunners }

func (a *AppRunnerService) MaxBatchSize() int { return 19 }

func (a *AppRunnerService) Nuke(identifiers []string) error {
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

	a.AppRunners = aws.StringValueSlice(identifiers)
	return a.AppRunners, nil
}

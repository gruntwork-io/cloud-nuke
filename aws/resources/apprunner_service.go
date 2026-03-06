package resources

import (
	"context"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AppRunnerAllowedRegions lists AWS regions where App Runner is supported.
// Reference: https://docs.aws.amazon.com/general/latest/gr/apprunner.html
var AppRunnerAllowedRegions = []string{
	"us-east-1", "us-east-2", "us-west-2", "ap-south-1", "ap-southeast-1", "ap-southeast-2",
	"ap-northeast-1", "eu-central-1", "eu-west-1", "eu-west-2", "eu-west-3",
}

// AppRunnerServiceAPI defines the interface for App Runner service operations.
type AppRunnerServiceAPI interface {
	DeleteService(ctx context.Context, params *apprunner.DeleteServiceInput, optFns ...func(*apprunner.Options)) (*apprunner.DeleteServiceOutput, error)
	ListServices(ctx context.Context, params *apprunner.ListServicesInput, optFns ...func(*apprunner.Options)) (*apprunner.ListServicesOutput, error)
}

// NewAppRunnerService creates a new App Runner service resource.
func NewAppRunnerService() AwsResource {
	return NewAwsResource(&resource.Resource[AppRunnerServiceAPI]{
		ResourceTypeName: "app-runner-service",
		BatchSize:        20,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[AppRunnerServiceAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = apprunner.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.AppRunnerService
		},
		Lister: listAppRunnerServices,
		Nuker:  resource.SimpleBatchDeleter(deleteAppRunnerService),
	})
}

// listAppRunnerServices retrieves all App Runner services that match the config filters.
func listAppRunnerServices(ctx context.Context, client AppRunnerServiceAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// Check if region supports App Runner
	if !slices.Contains(AppRunnerAllowedRegions, scope.Region) {
		logging.Debugf("Region %s is not allowed for App Runner", scope.Region)
		return nil, nil
	}

	var identifiers []*string
	paginator := apprunner.NewListServicesPaginator(client, &apprunner.ListServicesInput{
		MaxResults: aws.Int32(20),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, service := range page.ServiceSummaryList {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: service.ServiceName,
				Time: service.CreatedAt,
			}) {
				identifiers = append(identifiers, service.ServiceArn)
			}
		}
	}

	return identifiers, nil
}

// deleteAppRunnerService deletes a single App Runner service (asynchronous operation).
func deleteAppRunnerService(ctx context.Context, client AppRunnerServiceAPI, serviceArn *string) error {
	_, err := client.DeleteService(ctx, &apprunner.DeleteServiceInput{
		ServiceArn: serviceArn,
	})
	return err
}

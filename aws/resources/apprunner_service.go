package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AppRunnerServiceAPI defines the interface for AppRunner Service operations.
type AppRunnerServiceAPI interface {
	DeleteService(ctx context.Context, params *apprunner.DeleteServiceInput, optFns ...func(*apprunner.Options)) (*apprunner.DeleteServiceOutput, error)
	ListServices(ctx context.Context, params *apprunner.ListServicesInput, optFns ...func(*apprunner.Options)) (*apprunner.ListServicesOutput, error)
}

// NewAppRunnerService creates a new AppRunnerService resource using the generic resource pattern.
func NewAppRunnerService() AwsResource {
	return NewAwsResource(&resource.Resource[AppRunnerServiceAPI]{
		ResourceTypeName: "app-runner-service",
		BatchSize:        19,
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

// listAppRunnerServices retrieves all AppRunner Services that match the config filters.
func listAppRunnerServices(ctx context.Context, client AppRunnerServiceAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string
	paginator := apprunner.NewListServicesPaginator(client, &apprunner.ListServicesInput{
		MaxResults: aws.Int32(19),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, service := range output.ServiceSummaryList {
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

// deleteAppRunnerService deletes a single AppRunner Service.
func deleteAppRunnerService(ctx context.Context, client AppRunnerServiceAPI, serviceArn *string) error {
	_, err := client.DeleteService(ctx, &apprunner.DeleteServiceInput{
		ServiceArn: serviceArn,
	})
	return err
}

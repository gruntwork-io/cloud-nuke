package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// CodeDeployApplicationsAPI defines the interface for CodeDeploy operations.
type CodeDeployApplicationsAPI interface {
	ListApplications(ctx context.Context, params *codedeploy.ListApplicationsInput, optFns ...func(*codedeploy.Options)) (*codedeploy.ListApplicationsOutput, error)
	BatchGetApplications(ctx context.Context, params *codedeploy.BatchGetApplicationsInput, optFns ...func(*codedeploy.Options)) (*codedeploy.BatchGetApplicationsOutput, error)
	DeleteApplication(ctx context.Context, params *codedeploy.DeleteApplicationInput, optFns ...func(*codedeploy.Options)) (*codedeploy.DeleteApplicationOutput, error)
}

// NewCodeDeployApplications creates a new CodeDeployApplications resource.
func NewCodeDeployApplications() AwsResource {
	return NewAwsResource(&resource.Resource[CodeDeployApplicationsAPI]{
		ResourceTypeName: "codedeploy-application",
		BatchSize:        100,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[CodeDeployApplicationsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = codedeploy.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CodeDeployApplications
		},
		Lister: listCodeDeployApplications,
		Nuker:  resource.SimpleBatchDeleter(deleteCodeDeployApplication),
	})
}

// listCodeDeployApplications retrieves all CodeDeploy applications that match the config filters.
func listCodeDeployApplications(ctx context.Context, client CodeDeployApplicationsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	paginator := codedeploy.NewListApplicationsPaginator(client, &codedeploy.ListApplicationsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Filter by name first to reduce BatchGetApplications API calls
		var namesToCheck []string
		for _, name := range page.Applications {
			if cfg.ShouldInclude(config.ResourceValue{Name: aws.String(name)}) {
				namesToCheck = append(namesToCheck, name)
			}
		}

		if len(namesToCheck) == 0 {
			continue
		}

		// Get creation dates for this page's applications
		resp, err := client.BatchGetApplications(ctx, &codedeploy.BatchGetApplicationsInput{
			ApplicationNames: namesToCheck,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, app := range resp.ApplicationsInfo {
			if cfg.ShouldInclude(config.ResourceValue{Time: app.CreateTime}) {
				result = append(result, app.ApplicationName)
			}
		}
	}

	return result, nil
}

// deleteCodeDeployApplication deletes a single CodeDeploy application.
func deleteCodeDeployApplication(ctx context.Context, client CodeDeployApplicationsAPI, name *string) error {
	_, err := client.DeleteApplication(ctx, &codedeploy.DeleteApplicationInput{ApplicationName: name})
	return err
}

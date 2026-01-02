package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// EBApplicationsAPI defines the interface for Elastic Beanstalk operations.
type EBApplicationsAPI interface {
	DescribeApplications(ctx context.Context, params *elasticbeanstalk.DescribeApplicationsInput, optFns ...func(*elasticbeanstalk.Options)) (*elasticbeanstalk.DescribeApplicationsOutput, error)
	DeleteApplication(ctx context.Context, params *elasticbeanstalk.DeleteApplicationInput, optFns ...func(*elasticbeanstalk.Options)) (*elasticbeanstalk.DeleteApplicationOutput, error)
}

// NewEBApplications creates a new Elastic Beanstalk Applications resource using the generic resource pattern.
func NewEBApplications() AwsResource {
	return NewAwsResource(&resource.Resource[EBApplicationsAPI]{
		ResourceTypeName: "elastic-beanstalk",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EBApplicationsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = elasticbeanstalk.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ElasticBeanstalk
		},
		Lister: listEBApplications,
		Nuker:  resource.SimpleBatchDeleter(deleteEBApplication),
	})
}

// listEBApplications retrieves all Elastic Beanstalk applications that match the config filters.
func listEBApplications(ctx context.Context, client EBApplicationsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.DescribeApplications(ctx, &elasticbeanstalk.DescribeApplicationsInput{})
	if err != nil {
		return nil, err
	}

	var appNames []*string
	for _, app := range output.Applications {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: app.ApplicationName,
			Time: app.DateCreated,
		}) {
			appNames = append(appNames, app.ApplicationName)
		}
	}

	return appNames, nil
}

// deleteEBApplication deletes a single Elastic Beanstalk application.
func deleteEBApplication(ctx context.Context, client EBApplicationsAPI, appName *string) error {
	_, err := client.DeleteApplication(ctx, &elasticbeanstalk.DeleteApplicationInput{
		ApplicationName:     appName,
		TerminateEnvByForce: aws.Bool(true),
	})
	return err
}

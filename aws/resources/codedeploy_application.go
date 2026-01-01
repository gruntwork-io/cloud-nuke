package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// CodeDeployApplicationsAPI defines the interface for CodeDeploy operations.
type CodeDeployApplicationsAPI interface {
	ListApplications(ctx context.Context, params *codedeploy.ListApplicationsInput, optFns ...func(*codedeploy.Options)) (*codedeploy.ListApplicationsOutput, error)
	BatchGetApplications(ctx context.Context, params *codedeploy.BatchGetApplicationsInput, optFns ...func(*codedeploy.Options)) (*codedeploy.BatchGetApplicationsOutput, error)
	DeleteApplication(ctx context.Context, params *codedeploy.DeleteApplicationInput, optFns ...func(*codedeploy.Options)) (*codedeploy.DeleteApplicationOutput, error)
}

// NewCodeDeployApplications creates a new CodeDeployApplications resource using the generic resource pattern.
func NewCodeDeployApplications() AwsResource {
	return NewAwsResource(&resource.Resource[CodeDeployApplicationsAPI]{
		ResourceTypeName: "codedeploy-application",
		BatchSize:        100,
		InitClient: func(r *resource.Resource[CodeDeployApplicationsAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for CodeDeployApplications client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = codedeploy.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CodeDeployApplications
		},
		Lister: listCodeDeployApplications,
		Nuker:  resource.SimpleBatchDeleter(deleteCodeDeployApplication),
	})
}

// listCodeDeployApplications retrieves all CodeDeploy applications that match the config filters.
func listCodeDeployApplications(ctx context.Context, client CodeDeployApplicationsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var codeDeployApplicationsFilteredByName []string

	paginator := codedeploy.NewListApplicationsPaginator(client, &codedeploy.ListApplicationsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, application := range page.Applications {
			// Check if the CodeDeploy Application should be excluded by name as that information is available to us here.
			// CreationDate is not available in the ListApplications API call, so we can't filter by that here, but we do filter by it later.
			// By filtering the name here, we can reduce the number of BatchGetApplication API calls we have to make.
			if cfg.ShouldInclude(config.ResourceValue{Name: aws.String(application)}) {
				codeDeployApplicationsFilteredByName = append(codeDeployApplicationsFilteredByName, application)
			}
		}
	}

	// Check if the CodeDeploy Application should be excluded by CreationDate and return.
	// We have to do this after the ListApplicationsPages API call because CreationDate is not available in that call.
	return batchDescribeAndFilterCodeDeployApplications(ctx, client, codeDeployApplicationsFilteredByName, cfg)
}

// batchDescribeAndFilterCodeDeployApplications - Describe the CodeDeploy Applications and filter out the ones that should be excluded by CreationDate.
func batchDescribeAndFilterCodeDeployApplications(ctx context.Context, client CodeDeployApplicationsAPI, identifiers []string, cfg config.ResourceType) ([]*string, error) {
	// BatchGetApplications can only take 100 identifiers at a time, so we have to break up the identifiers into chunks of 100.
	batchSize := 100
	var applicationNames []*string

	for {
		// if there are no identifiers left, then break out of the loop
		if len(identifiers) == 0 {
			break
		}

		// if the batch size is larger than the number of identifiers left, then set the batch size to the number of identifiers left
		if len(identifiers) < batchSize {
			batchSize = len(identifiers)
		}

		// get the next batch of identifiers
		batch := identifiers[:batchSize]
		// then using that batch of identifiers, get the applicationsinfo
		resp, err := client.BatchGetApplications(
			ctx,
			&codedeploy.BatchGetApplicationsInput{ApplicationNames: batch},
		)
		if err != nil {
			return nil, err
		}

		// for each applicationsinfo, check if it should be excluded by creation date
		for j := range resp.ApplicationsInfo {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: resp.ApplicationsInfo[j].CreateTime,
			}) {
				applicationNames = append(applicationNames, resp.ApplicationsInfo[j].ApplicationName)
			}
		}

		// reduce the identifiers by the batch size we just processed, note that the slice header is mutated here
		identifiers = identifiers[batchSize:]
	}

	return applicationNames, nil
}

// deleteCodeDeployApplication deletes a single CodeDeploy Application.
func deleteCodeDeployApplication(ctx context.Context, client CodeDeployApplicationsAPI, identifier *string) error {
	_, err := client.DeleteApplication(ctx, &codedeploy.DeleteApplicationInput{ApplicationName: identifier})
	return err
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// ECRAPI defines the interface for ECR operations.
type ECRAPI interface {
	DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	DeleteRepository(ctx context.Context, params *ecr.DeleteRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.DeleteRepositoryOutput, error)
	ListTagsForResource(ctx context.Context, params *ecr.ListTagsForResourceInput, optFns ...func(*ecr.Options)) (*ecr.ListTagsForResourceOutput, error)
}

// NewECR creates a new ECR resource using the generic resource pattern.
func NewECR() AwsResource {
	return NewAwsResource(&resource.Resource[ECRAPI]{
		ResourceTypeName: "ecr",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ECRAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ecr.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ECRRepository
		},
		Lister: listECRRepositories,
		Nuker:  resource.SimpleBatchDeleter(deleteECRRepository),
	})
}

// listECRRepositories retrieves all ECR repositories that match the config filters.
func listECRRepositories(ctx context.Context, client ECRAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var repositoryNames []*string

	paginator := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, repository := range page.Repositories {
			tagsOutput, err := client.ListTagsForResource(ctx, &ecr.ListTagsForResourceInput{
				ResourceArn: repository.RepositoryArn,
			})
			if err != nil {
				logging.Debugf("[Failed] Unable to fetch tags for ECR repository %s: %s", aws.ToString(repository.RepositoryName), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Time: repository.CreatedAt,
				Name: repository.RepositoryName,
				Tags: util.ConvertECRTagsToMap(tagsOutput.Tags),
			}) {
				repositoryNames = append(repositoryNames, repository.RepositoryName)
			}
		}
	}

	return repositoryNames, nil
}

// deleteECRRepository deletes a single ECR repository.
func deleteECRRepository(ctx context.Context, client ECRAPI, repositoryName *string) error {
	_, err := client.DeleteRepository(ctx, &ecr.DeleteRepositoryInput{
		Force:          true,
		RepositoryName: repositoryName,
	})
	return err
}

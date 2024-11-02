package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ECRAPI interface {
	DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	DeleteRepository(ctx context.Context, params *ecr.DeleteRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.DeleteRepositoryOutput, error)
}

type ECR struct {
	BaseAwsResource
	Client          ECRAPI
	Region          string
	RepositoryNames []string
}

func (registry *ECR) InitV2(cfg aws.Config) {
	registry.Client = ecr.NewFromConfig(cfg)
}

func (registry *ECR) IsUsingV2() bool { return true }

func (registry *ECR) ResourceName() string {
	return "ecr"
}

func (registry *ECR) ResourceIdentifiers() []string {
	return registry.RepositoryNames
}

func (registry *ECR) MaxBatchSize() int {
	return 50
}

func (registry *ECR) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ECRRepository
}

func (registry *ECR) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := registry.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	registry.RepositoryNames = aws.ToStringSlice(identifiers)
	return registry.RepositoryNames, nil
}

func (registry *ECR) Nuke(identifiers []string) error {
	if err := registry.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

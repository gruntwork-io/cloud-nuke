package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ElasticFileSystemAPI interface {
	DeleteAccessPoint(ctx context.Context, params *efs.DeleteAccessPointInput, optFns ...func(*efs.Options)) (*efs.DeleteAccessPointOutput, error)
	DeleteFileSystem(ctx context.Context, params *efs.DeleteFileSystemInput, optFns ...func(*efs.Options)) (*efs.DeleteFileSystemOutput, error)
	DeleteMountTarget(ctx context.Context, params *efs.DeleteMountTargetInput, optFns ...func(*efs.Options)) (*efs.DeleteMountTargetOutput, error)
	DescribeAccessPoints(ctx context.Context, params *efs.DescribeAccessPointsInput, optFns ...func(*efs.Options)) (*efs.DescribeAccessPointsOutput, error)
	DescribeMountTargets(ctx context.Context, params *efs.DescribeMountTargetsInput, optFns ...func(*efs.Options)) (*efs.DescribeMountTargetsOutput, error)
	DescribeFileSystems(ctx context.Context, params *efs.DescribeFileSystemsInput, optFns ...func(*efs.Options)) (*efs.DescribeFileSystemsOutput, error)
}

type ElasticFileSystem struct {
	BaseAwsResource
	Client ElasticFileSystemAPI
	Region string
	Ids    []string
}

func (ef *ElasticFileSystem) InitV2(cfg aws.Config) {
	ef.Client = efs.NewFromConfig(cfg)
}

func (ef *ElasticFileSystem) ResourceName() string {
	return "efs"
}

func (ef *ElasticFileSystem) ResourceIdentifiers() []string {
	return ef.Ids
}

func (ef *ElasticFileSystem) MaxBatchSize() int {
	return 10
}

func (ef *ElasticFileSystem) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ElasticFileSystem
}

func (ef *ElasticFileSystem) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ef.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ef.Ids = aws.ToStringSlice(identifiers)
	return ef.Ids, nil
}

func (ef *ElasticFileSystem) Nuke(identifiers []string) error {
	if err := ef.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// custom errors

type TooManyElasticFileSystemsErr struct{}

func (err TooManyElasticFileSystemsErr) Error() string {
	return "Too many Elastic FileSystems requested at once."
}

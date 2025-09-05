package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudFormationStacksAPI interface {
	ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
	DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

type CloudFormationStacks struct {
	BaseAwsResource
	Client     CloudFormationStacksAPI
	Region     string
	StackNames []string
}

func (cfs *CloudFormationStacks) Init(cfg aws.Config) {
	cfs.Client = cloudformation.NewFromConfig(cfg)
}

func (cfs *CloudFormationStacks) ResourceName() string {
	return "cloudformation-stack"
}

func (cfs *CloudFormationStacks) ResourceIdentifiers() []string {
	return cfs.StackNames
}

func (cfs *CloudFormationStacks) MaxBatchSize() int {
	return 49
}

func (cfs *CloudFormationStacks) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudFormationStack
}

func (cfs *CloudFormationStacks) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cfs.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cfs.StackNames = aws.ToStringSlice(identifiers)
	return cfs.StackNames, nil
}

func (cfs *CloudFormationStacks) Nuke(identifiers []string) error {
	if err := cfs.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
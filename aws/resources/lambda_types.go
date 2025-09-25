package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type LambdaFunctionsAPI interface {
	DeleteFunction(ctx context.Context, params *lambda.DeleteFunctionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteFunctionOutput, error)
	ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
	ListTags(ctx context.Context, params *lambda.ListTagsInput, optFns ...func(*lambda.Options)) (*lambda.ListTagsOutput, error)
}

type LambdaFunctions struct {
	BaseAwsResource
	Client              LambdaFunctionsAPI
	Region              string
	LambdaFunctionNames []string
}

func (lf *LambdaFunctions) Init(cfg aws.Config) {
	lf.Client = lambda.NewFromConfig(cfg)
}

func (lf *LambdaFunctions) ResourceName() string {
	return "lambda"
}

// ResourceIdentifiers - The names of the lambda functions
func (lf *LambdaFunctions) ResourceIdentifiers() []string {
	return lf.LambdaFunctionNames
}

func (lf *LambdaFunctions) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (lf *LambdaFunctions) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.LambdaFunction
}

func (lf *LambdaFunctions) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := lf.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	lf.LambdaFunctionNames = aws.ToStringSlice(identifiers)
	return lf.LambdaFunctionNames, nil
}

// Nuke - nuke 'em all!!!
func (lf *LambdaFunctions) Nuke(identifiers []string) error {
	if err := lf.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type LambdaDeleteError struct {
	name string
}

func (e LambdaDeleteError) Error() string {
	return "Lambda Function:" + e.name + "was not deleted"
}

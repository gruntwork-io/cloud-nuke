package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type LambdaLayersAPI interface {
	DeleteLayerVersion(ctx context.Context, params *lambda.DeleteLayerVersionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteLayerVersionOutput, error)
	ListLayers(ctx context.Context, params *lambda.ListLayersInput, optFns ...func(*lambda.Options)) (*lambda.ListLayersOutput, error)
	ListLayerVersions(ctx context.Context, params *lambda.ListLayerVersionsInput, optFns ...func(*lambda.Options)) (*lambda.ListLayerVersionsOutput, error)
}

type LambdaLayers struct {
	BaseAwsResource
	Client              LambdaLayersAPI
	Region              string
	LambdaFunctionNames []string
}

func (ll *LambdaLayers) InitV2(cfg aws.Config) {
	ll.Client = lambda.NewFromConfig(cfg)
}

func (ll *LambdaLayers) IsUsingV2() bool { return true }

func (ll *LambdaLayers) ResourceName() string {
	return "lambda_layer"
}

// ResourceIdentifiers - The names of the lambda functions
func (ll *LambdaLayers) ResourceIdentifiers() []string {
	return ll.LambdaFunctionNames
}

func (ll *LambdaLayers) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (ll *LambdaLayers) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.LambdaLayer
}

func (ll *LambdaLayers) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ll.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ll.LambdaFunctionNames = aws.ToStringSlice(identifiers)
	return ll.LambdaFunctionNames, nil
}

// Nuke - nuke 'em all!!!
func (ll *LambdaLayers) Nuke(identifiers []string) error {
	if err := ll.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type LambdaVersionDeleteError struct {
	name string
}

func (e LambdaVersionDeleteError) Error() string {
	return "Lambda Function:" + e.name + "was not deleted"
}

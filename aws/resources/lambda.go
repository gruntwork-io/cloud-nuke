package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// LambdaFunctionsAPI defines the interface for Lambda operations.
type LambdaFunctionsAPI interface {
	DeleteFunction(ctx context.Context, params *lambda.DeleteFunctionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteFunctionOutput, error)
	ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
	ListTags(ctx context.Context, params *lambda.ListTagsInput, optFns ...func(*lambda.Options)) (*lambda.ListTagsOutput, error)
}

// NewLambdaFunctions creates a new Lambda Functions resource using the generic resource pattern.
func NewLambdaFunctions() AwsResource {
	return NewAwsResource(&resource.Resource[LambdaFunctionsAPI]{
		ResourceTypeName: "lambda",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[LambdaFunctionsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = lambda.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.LambdaFunction
		},
		Lister: listLambdaFunctions,
		Nuker:  resource.SimpleBatchDeleter(deleteLambdaFunction),
	})
}

// listLambdaFunctions retrieves all Lambda functions that match the config filters.
func listLambdaFunctions(ctx context.Context, client LambdaFunctionsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string

	paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, fn := range page.Functions {
			if shouldIncludeLambdaFunction(ctx, client, &fn, cfg) {
				names = append(names, fn.FunctionName)
			}
		}
	}

	return names, nil
}

// shouldIncludeLambdaFunction determines if a Lambda function should be included for deletion.
func shouldIncludeLambdaFunction(ctx context.Context, client LambdaFunctionsAPI, lambdaFn *types.FunctionConfiguration, cfg config.ResourceType) bool {
	if lambdaFn == nil {
		return false
	}

	fnLastModified := aws.ToString(lambdaFn.LastModified)
	fnName := lambdaFn.FunctionName
	layout := "2006-01-02T15:04:05.000+0000"
	lastModifiedDateTime, err := time.Parse(layout, fnLastModified)
	if err != nil {
		logging.Debugf("Could not parse last modified timestamp (%s) of Lambda function %s. Excluding from delete.", fnLastModified, *fnName)
		return false
	}

	params := &lambda.ListTagsInput{
		Resource: lambdaFn.FunctionArn,
	}
	tagsOutput, err := client.ListTags(ctx, params)
	if err != nil {
		logging.Errorf("failed to list tags for %s: %s", aws.ToString(lambdaFn.FunctionArn), err)
	}

	var tags map[string]string
	if tagsOutput != nil {
		tags = tagsOutput.Tags
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Time: &lastModifiedDateTime,
		Name: fnName,
		Tags: tags,
	})
}

// deleteLambdaFunction deletes a single Lambda function.
func deleteLambdaFunction(ctx context.Context, client LambdaFunctionsAPI, name *string) error {
	_, err := client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: name,
	})
	return err
}

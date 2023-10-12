package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
)

func (lf *LambdaFunctionVersions) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var names []*string

	err := lf.Client.ListFunctionsPages(
		&lambda.ListFunctionsInput{}, func(page *lambda.ListFunctionsOutput, lastPage bool) bool {
			for _, lambda := range page.Functions {
				if lf.shouldInclude(lambda, configObj) {
					names = append(names, lambda.FunctionName)
				}
			}

			return !lastPage
		})

	if err != nil {
		return nil, err
	}

	return names, nil
}

func (lf *LambdaFunctionVersions) shouldInclude(lambdaFn *lambda.FunctionConfiguration, configObj config.Config) bool {
	if lambdaFn == nil {
		return false
	}

	fnLastModified := aws.StringValue(lambdaFn.LastModified)
	fnName := lambdaFn.FunctionName
	layout := "2006-01-02T15:04:05.000+0000"
	lastModifiedDateTime, err := time.Parse(layout, fnLastModified)
	if err != nil {
		logging.Logger.Debugf("Could not parse last modified timestamp (%s) of Lambda function %s. Excluding from delete.", fnLastModified, *fnName)
		return false
	}

	return configObj.LambdaFunction.ShouldInclude(config.ResourceValue{
		Time: &lastModifiedDateTime,
		Name: fnName,
	})
}

package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllLambdaFunctions(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := lambda.New(session)

	result, err := svc.ListFunctions(nil)

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, lambda := range result.Functions {
		if shouldIncludeLambdaFunction(lambda, excludeAfter, configObj) {
			names = append(names, lambda.FunctionName)
		}
	}

	return names, nil
}

func shouldIncludeLambdaFunction(lambdaFn *lambda.FunctionConfiguration, excludeAfter time.Time, configObj config.Config) bool {
	if lambdaFn == nil {
		return false
	}

	fnLastModified := aws.StringValue(lambdaFn.LastModified)
	fnName := aws.StringValue(lambdaFn.FunctionName)

	layout := "2006-01-02T15:04:05.000+0000"
	lastModifiedDateTime, err := time.Parse(layout, fnLastModified)
	if err != nil {
		logging.Logger.Warnf("Could not parse last modified timestamp (%s) of Lambda function %s. Excluding from delete.", fnLastModified, fnName)
		return false
	}

	if excludeAfter.Before(lastModifiedDateTime) {
		return false
	}

	return config.ShouldInclude(
		fnName,
		configObj.LambdaFunction.IncludeRule.NamesRegExp,
		configObj.LambdaFunction.ExcludeRule.NamesRegExp,
	)
}

func nukeAllLambdaFunctions(session *session.Session, names []*string) error {
	svc := lambda.New(session)

	if len(names) == 0 {
		logging.Logger.Infof("No Lambda Functions to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all Lambda Functions in region %s", *session.Config.Region)
	deletedNames := []*string{}

	for _, name := range names {
		params := &lambda.DeleteFunctionInput{
			FunctionName: name,
		}

		_, err := svc.DeleteFunction(params)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Infof("Deleted Lambda Function: %s", awsgo.StringValue(name))
		}
	}

	logging.Logger.Infof("[OK] %d Lambda Function(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}

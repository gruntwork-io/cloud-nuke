package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllLambdaFunctions(session *session.Session, excludeAfter time.Time, configObj config.Config, batchSize int) ([]*string, error) {
	svc := lambda.New(session)

	var result []*lambda.FunctionConfiguration

	var next *string = nil
	for {
		list, err := svc.ListFunctions(&lambda.ListFunctionsInput{
			Marker:   next,
			MaxItems: awsgo.Int64(int64(batchSize)),
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		result = append(result, list.Functions...)
		if list.NextMarker == nil || len(*list.NextMarker) == 0 {
			break
		}
		next = list.NextMarker
	}

	var names []*string

	for _, lambda := range result {
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
		logging.Logger.Debugf("Could not parse last modified timestamp (%s) of Lambda function %s. Excluding from delete.", fnLastModified, fnName)
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
		logging.Logger.Debugf("No Lambda Functions to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Lambda Functions in region %s", *session.Config.Region)
	deletedNames := []*string{}

	for _, name := range names {
		params := &lambda.DeleteFunctionInput{
			FunctionName: name,
		}

		_, err := svc.DeleteFunction(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(name),
			ResourceType: "Lambda function",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *name, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Lambda Function",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted Lambda Function: %s", awsgo.StringValue(name))
		}
	}

	logging.Logger.Debugf("[OK] %d Lambda Function(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}

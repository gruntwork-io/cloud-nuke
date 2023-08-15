package resources

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func (lf *LambdaFunctions) getAll(configObj config.Config) ([]*string, error) {
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

func (lf *LambdaFunctions) shouldInclude(lambdaFn *lambda.FunctionConfiguration, configObj config.Config) bool {
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

func (lf *LambdaFunctions) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Logger.Debugf("No Lambda Functions to nuke in region %s", lf.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Lambda Functions in region %s", lf.Region)
	deletedNames := []*string{}

	for _, name := range names {
		params := &lambda.DeleteFunctionInput{
			FunctionName: name,
		}

		_, err := lf.Client.DeleteFunction(params)

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
				"region": lf.Region,
			})
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted Lambda Function: %s", awsgo.StringValue(name))
		}
	}

	logging.Logger.Debugf("[OK] %d Lambda Function(s) deleted in %s", len(deletedNames), lf.Region)
	return nil
}

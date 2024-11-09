package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (lf *LambdaFunctions) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var names []*string

	paginator := lambda.NewListFunctionsPaginator(lf.Client, &lambda.ListFunctionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, name := range page.Functions {
			if lf.shouldInclude(&name, configObj) {
				names = append(names, name.FunctionName)
			}
		}
	}

	return names, nil
}

func (lf *LambdaFunctions) shouldInclude(lambdaFn *types.FunctionConfiguration, configObj config.Config) bool {
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

	return configObj.LambdaFunction.ShouldInclude(config.ResourceValue{
		Time: &lastModifiedDateTime,
		Name: fnName,
	})
}

func (lf *LambdaFunctions) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No Lambda Functions to nuke in region %s", lf.Region)
		return nil
	}

	logging.Debugf("Deleting all Lambda Functions in region %s", lf.Region)
	var deletedNames []*string

	for _, name := range names {
		params := &lambda.DeleteFunctionInput{
			FunctionName: name,
		}

		_, err := lf.Client.DeleteFunction(lf.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(name),
			ResourceType: "Lambda function",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted Lambda Function: %s", aws.ToString(name))
		}
	}

	logging.Debugf("[OK] %d Lambda Function(s) deleted in %s", len(deletedNames), lf.Region)
	return nil
}

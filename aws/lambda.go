package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllLambdaFunctions(session *session.Session, excludeAfter time.Time, batchSize int) ([]*string, error) {
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
		layout := "2006-01-02T15:04:05.000+0000"
		lastModifiedDateTime, err := time.Parse(layout, *lambda.LastModified)
		if err != nil {
			return nil, err
		}

		if lambda.LastModified != nil && excludeAfter.After(lastModifiedDateTime) {
			names = append(names, lambda.FunctionName)
		}
	}

	return names, nil
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

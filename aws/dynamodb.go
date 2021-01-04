package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAllDynamoTables(session *session.Session) ([]*string, error) {
	var TableNames []*string
	svc := dynamodb.New(session)

	for {
		result, err := svc.ListTables(&dynamodb.ListTablesInput{})
		if err != nil {
			//
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Error() {
				case dynamodb.ErrCodeInternalServerError:
					return nil, errors.WithStackTrace(aerr)
				default:
					return nil, errors.WithStackTrace(aerr)
				}
			} else {
				return nil, errors.WithStackTrace(aerr)
			}
		}
		for _, table := range result.TableNames {
			TableNames = append(TableNames, table)
		}
		return TableNames, nil

	}

}

func nukeAllDynamoDBTables(session *session.Session, tables []*string) error {
	svc := dynamodb.New(session)
	if len(tables) == 0 {
		logging.Logger.Infof("No EBS volumes to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all EBS volumes in region %s", *session.Config.Region)
	for _, table := range tables {

		input := &dynamodb.DeleteTableInput{
			TableName: aws.String(*table),
		}
		_, err := svc.DeleteTable(input)
		if err != nil {
			//
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Error() {
				case dynamodb.ErrCodeInternalServerError:
					return errors.WithStackTrace(aerr)
				default:
					return errors.WithStackTrace(aerr)
				}
			} else {
				return errors.WithStackTrace(aerr)
			}
		}

	}
	return nil

}

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)




func getAllDynamoTables(session *session.Session) ([]*string, error) {
	var DynamoTableNames []*string
	svc := dynamodb.New(session)
	input := &dynamodb.ListTablesInput{}

	for {
		result, err := svc.ListTables(input)
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
			DynamoTableNames = append(DynamoTableNames, table)
		}
		return DynamoTableNames, nil

	}

}

func nukeAllDynamoDBTables(session *session.Session, tables []*string) error {
	svc := dynamodb.New(session)
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
					return  errors.WithStackTrace(aerr)
				default:
					return  errors.WithStackTrace(aerr)
				}
			} else {
				return  errors.WithStackTrace(aerr)
			}
		}

	}
	return nil


}

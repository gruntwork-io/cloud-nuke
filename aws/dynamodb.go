package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"log"
	"time"
)

func getAllDynamoTables(session *session.Session, excludeAfter time.Time, db DynamoDB) ([]*string, error) {
	var tableNames []*string
	svc := dynamodb.New(session)
	// Run count is used for pagination if the list tables exceeds max value
	// Tells loop to rerun
	var runCount = 1
	for runCount > 0 {
		result, err := svc.ListTables(&dynamodb.ListTablesInput{Limit: aws.Int64(int64(DynamoDB.MaxBatchSize(db)))})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Error() {
				case dynamodb.ErrCodeInternalServerError:
					return nil, errors.WithStackTrace(aerr)
				default:
					return nil, errors.WithStackTrace(aerr)
				}
			}
		}

		tableLen := len(result.TableNames)
		// Check table length if it matches the max value add 1 to rerun
		if tableLen == DynamoDB.MaxBatchSize(db) {
			// Tell the user that this will be run twice due to max tables detected
			logging.Logger.Infof("The tables detected exceed the 100. Running more than once")
			// Adds one to the count as it will = 2 runs at least until this loops again to check if it's another max.
			runCount += 1
		}
		for _, table := range result.TableNames {

			responseDescription, err := svc.DescribeTable(&dynamodb.DescribeTableInput{TableName: table})
			if err != nil {
				log.Fatalf("There was an error describing table: %v\n", err)
			}
			// This is used in case of a nil so null pointers don't occur
			if responseDescription.Table.CreationDateTime == nil {
				break
			}
			if excludeAfter.After(*responseDescription.Table.CreationDateTime) {

				tableNames = append(tableNames, table)
			}
		}
		// Remove 1 from the counter if it's one run the loop will end as runCount will = 0
		runCount -= 1
		// Empty the slice for reuse.
		tableNames = nil
	}


	return tableNames, nil

}

func nukeAllDynamoDBTables(session *session.Session, tables []*string) error {
	svc := dynamodb.New(session)
	if len(tables) == 0 {
		logging.Logger.Infof("No DynamoDB tables to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all DynamoDB tables in region %s", *session.Config.Region)
	for _, table := range tables {

		input := &dynamodb.DeleteTableInput{
			TableName: aws.String(*table),
		}
		_, err := svc.DeleteTable(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Error() {
				case dynamodb.ErrCodeInternalServerError:
					return errors.WithStackTrace(aerr)
				default:
					return errors.WithStackTrace(aerr)
				}
			}
		}
	}
	return nil
}

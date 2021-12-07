package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"log"
	"time"
)

func getAllDynamoTables(session *session.Session, excludeAfter time.Time, configObj config.Config, db DynamoDB) ([]*string, error) {
	var tableNames []*string
	svc := dynamodb.New(session)
	
	var lastTableName *string
	// Run count is used for pagination if the list tables exceeds max value
	// Tells loop to rerun
	var PaginationRunCount = 1
	for PaginationRunCount > 0 {
		result, err := svc.ListTables(&dynamodb.ListTablesInput{ExclusiveStartTableName: lastTableName, Limit: aws.Int64(int64(DynamoDB.MaxBatchSize(db)))})
		
		lastTableName = result.LastEvaluatedTableName
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
			PaginationRunCount += 1
		}
		for _, table := range result.TableNames {
			responseDescription, err := svc.DescribeTable(&dynamodb.DescribeTableInput{TableName: table})
			if err != nil {
				log.Fatalf("There was an error describing table: %v\n", err)
			}

			if shouldIncludeTable(responseDescription.Table, excludeAfter, configObj) {
				tableNames = append(tableNames, table)
			}
		}
		// Remove 1 from the counter if it's one run the loop will end as PaginationRunCount will = 0
		PaginationRunCount -= 1
	}
	return tableNames, nil
}

func shouldIncludeTable(table *dynamodb.TableDescription, excludeAfter time.Time, configObj config.Config) bool{
	if table == nil {
		return false
	}

	if table.CreationDateTime != nil && excludeAfter.Before(*table.CreationDateTime) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(table.TableName),
		configObj.DynamoDB.IncludeRule.NamesRegExp,
		configObj.DynamoDB.ExcludeRule.NamesRegExp,
	)
}

func nukeAllDynamoDBTables(session *session.Session, tables []*string) error {
	svc := dynamodb.New(session)
	if len(tables) == 0 {
		logging.Logger.Infof("No DynamoDB tables to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all DynamoDB tables in region %s", *session.Config.Region)
	for _, table := range tables {

		log.Printf("Deleting table %v", aws.String(*table))
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
	log.Println("Returning nil error")
	return nil
}

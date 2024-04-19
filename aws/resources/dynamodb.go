package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"log"
)

func (ddb *DynamoDB) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var tableNames []*string

	err := ddb.Client.ListTablesPages(
		&dynamodb.ListTablesInput{}, func(page *dynamodb.ListTablesOutput, lastPage bool) bool {
			for _, table := range page.TableNames {
				tableDetail, err := ddb.Client.DescribeTable(&dynamodb.DescribeTableInput{TableName: table})
				if err != nil {
					log.Fatalf("There was an error describing table: %v\n", err)
				}

				if configObj.DynamoDB.ShouldInclude(config.ResourceValue{
					Time: tableDetail.Table.CreationDateTime,
					Name: tableDetail.Table.TableName,
				}) {
					tableNames = append(tableNames, table)
				}
			}

			return !lastPage
		})

	if err != nil {
		return nil, err
	}

	return tableNames, nil
}

func (ddb *DynamoDB) nukeAll(tables []*string) error {
	if len(tables) == 0 {
		logging.Debugf("No DynamoDB tables to nuke in region %s", ddb.Region)
		return nil
	}

	logging.Debugf("Deleting all DynamoDB tables in region %s", ddb.Region)
	for _, table := range tables {

		input := &dynamodb.DeleteTableInput{
			TableName: aws.String(*table),
		}
		_, err := ddb.Client.DeleteTable(input)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(table),
			ResourceType: "DynamoDB Table",
			Error:        err,
		}
		report.Record(e)

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

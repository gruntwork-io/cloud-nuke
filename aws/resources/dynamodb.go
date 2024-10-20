package resources

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func (ddb *DynamoDB) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var tableNames []*string

	paginator := dynamodb.NewListTablesPaginator(ddb.Client, &dynamodb.ListTablesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, err
		}

		for _, table := range page.TableNames {
			tableDetail, errPage := ddb.Client.DescribeTable(ddb.Context, &dynamodb.DescribeTableInput{TableName: aws.String(table)})
			if errPage != nil {
				log.Fatalf("There was an error describing table: %v\n", errPage)
			}

			if configObj.DynamoDB.ShouldInclude(config.ResourceValue{
				Time: tableDetail.Table.CreationDateTime,
				Name: tableDetail.Table.TableName,
			}) {
				tableNames = append(tableNames, aws.String(table))
			}
		}
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
		_, err := ddb.Client.DeleteTable(ddb.Context, input)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(table),
			ResourceType: "DynamoDB Table",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			return errors.WithStackTrace(err)
		}
	}
	return nil
}

package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// DynamoDBAPI defines the interface for DynamoDB operations.
type DynamoDBAPI interface {
	ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	UpdateTable(ctx context.Context, params *dynamodb.UpdateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateTableOutput, error)
	DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error)
}

// NewDynamoDB creates a new DynamoDB resource using the generic resource pattern.
func NewDynamoDB() AwsResource {
	return NewAwsResource(&resource.Resource[DynamoDBAPI]{
		ResourceTypeName: "dynamodb",
		BatchSize:        DefaultBatchSize, // Tentative batch size to ensure AWS doesn't throttle
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DynamoDBAPI], cfg aws.Config) {
			r.Client = dynamodb.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DynamoDB
		},
		Lister: listDynamoDBTables,
		Nuker:  resource.SequentialDeleteThenWaitAll(deleteDynamoDBTable, waitForDynamoDBTablesDeleted),
	})
}

// listDynamoDBTables retrieves all DynamoDB tables that match the config filters.
func listDynamoDBTables(ctx context.Context, client DynamoDBAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var tableNames []*string

	paginator := dynamodb.NewListTablesPaginator(client, &dynamodb.ListTablesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, table := range page.TableNames {
			tableDetail, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Time: tableDetail.Table.CreationDateTime,
				Name: tableDetail.Table.TableName,
			}) {
				tableNames = append(tableNames, aws.String(table))
			}
		}
	}

	return tableNames, nil
}

// deleteDynamoDBTable disables deletion protection and deletes a DynamoDB table.
func deleteDynamoDBTable(ctx context.Context, client DynamoDBAPI, tableName *string) error {
	// Disable deletion protection if enabled
	if _, err := client.UpdateTable(ctx, &dynamodb.UpdateTableInput{
		TableName:                 tableName,
		DeletionProtectionEnabled: aws.Bool(false),
	}); err != nil {
		logging.Warnf("[Failed] to disable deletion protection for %s: %s", aws.ToString(tableName), err)
	}

	_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: tableName,
	})
	return errors.WithStackTrace(err)
}

// waitForDynamoDBTablesDeleted waits for all specified DynamoDB tables to be deleted.
func waitForDynamoDBTablesDeleted(ctx context.Context, client DynamoDBAPI, ids []string) error {
	waiter := dynamodb.NewTableNotExistsWaiter(client)
	for _, id := range ids {
		if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(id),
		}, 10*time.Minute); err != nil {
			return err
		}
	}
	return nil
}

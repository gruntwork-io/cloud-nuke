package resources

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// DynamoDBAPI defines the interface for DynamoDB operations.
type DynamoDBAPI interface {
	ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error)
}

// NewDynamoDB creates a new DynamoDB resource using the generic resource pattern.
func NewDynamoDB() AwsResource {
	return NewAwsResource(&resource.Resource[DynamoDBAPI]{
		ResourceTypeName: "dynamodb",
		BatchSize:        49, // Tentative batch size to ensure AWS doesn't throttle
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DynamoDBAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = dynamodb.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DynamoDB
		},
		Lister: listDynamoDBTables,
		Nuker:  resource.SimpleBatchDeleter(deleteDynamoDBTable),
	})
}

// listDynamoDBTables retrieves all DynamoDB tables that match the config filters.
func listDynamoDBTables(ctx context.Context, client DynamoDBAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var tableNames []*string

	paginator := dynamodb.NewListTablesPaginator(client, &dynamodb.ListTablesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, table := range page.TableNames {
			tableDetail, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)})
			if err != nil {
				log.Fatalf("There was an error describing table: %v\n", err)
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

// deleteDynamoDBTable deletes a single DynamoDB table.
func deleteDynamoDBTable(ctx context.Context, client DynamoDBAPI, tableName *string) error {
	_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: tableName,
	})
	return err
}

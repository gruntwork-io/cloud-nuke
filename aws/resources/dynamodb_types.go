package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type DynamoDBAPI interface {
	ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error)
}

type DynamoDB struct {
	BaseAwsResource
	Client           DynamoDBAPI
	Region           string
	DynamoTableNames []string
}

func (ddb *DynamoDB) InitV2(cfg aws.Config) {
	ddb.Client = dynamodb.NewFromConfig(cfg)
}

func (ddb *DynamoDB) ResourceName() string {
	return "dynamodb"
}

func (ddb *DynamoDB) ResourceIdentifiers() []string {
	return ddb.DynamoTableNames
}

func (ddb *DynamoDB) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (ddb *DynamoDB) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DynamoDB
}

func (ddb *DynamoDB) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ddb.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ddb.DynamoTableNames = aws.ToStringSlice(identifiers)
	return ddb.DynamoTableNames, nil
}

// Nuke - nuke all Dynamo DB Tables
func (ddb *DynamoDB) Nuke(identifiers []string) error {
	if err := ddb.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

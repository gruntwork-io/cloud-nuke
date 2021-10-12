package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"testing"
	"time"
)

func createTestDynamoTables(t *testing.T, tableName string) {
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(awsSession)
	// THE INFORMATION TO CREATE THE TABLE
	input := &dynamodb.CreateTableInput{
		TableName: &tableName,
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Nuke" + string(rune(1))),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("TypeofNuke" + string(rune(1))),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Nuke" + string(rune(1))),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("TypeofNuke" + string(rune(1))),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	}
	// CREATING THE TABLE FROM THE INPUT
	_, err := svc.CreateTable(input)
	require.NoError(t, err)

}

func getTableStatus(TableName string) *string {
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(awsSession)

	tableInput := &dynamodb.DescribeTableInput{TableName: &TableName}

	result, err := svc.DescribeTable(tableInput)
	if err != nil {
		log.Fatalf("There was an error describing tables %v", err)
	}

	return result.Table.TableStatus

}

func TestGetTablesDynamo(t *testing.T) {
	t.Parallel()
	region, err := getRandomRegion()
	require.NoError(t, err)
	db := DynamoDB{}
	awsSession, err := session.NewSession(&aws.Config{
		Region: &region,
	})
	require.NoError(t, err)
	getAllDynamoTables(awsSession, time.Now().Add(1*time.Hour*-1), db)
}

func TestNukeAllDynamoDBTables(t *testing.T) {
	t.Parallel()
	db := DynamoDB{}

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	tableName := "cloud-nuke-test-" + util.UniqueID()
	defer nukeAllDynamoDBTables(awsSession, []*string{&tableName})
	createTestDynamoTables(t, tableName)
	COUNTER := 0
	for COUNTER <= 1 {
		tableStatus := getTableStatus(tableName)
		if *tableStatus == "ACTIVE" {
			COUNTER += 1
			log.Printf("Created a table: %v\n", tableName)
		} else {
			log.Printf("Table not ready yet: %v", tableName)
		}
	}
	nukeErr := nukeAllDynamoDBTables(awsSession, []*string{&tableName})
	require.NoError(t, nukeErr)

	tables, err := getAllDynamoTables(awsSession, time.Now().Add(1*time.Hour*-1), db)
	require.NoError(t, err)

	for _, table := range tables {
		if tableName == *table {
			assert.Fail(t, errors.WithStackTrace(err).Error())
		}
	}
}

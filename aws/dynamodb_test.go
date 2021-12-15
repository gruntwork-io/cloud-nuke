package aws

import (
	"log"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestDynamoTables(t *testing.T, tableName, region string) {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

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
	_, err = svc.CreateTable(input)
	require.NoError(t, err)

}

func getTableStatus(TableName string, region string) *string {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	svc := dynamodb.New(awsSession)

	tableInput := &dynamodb.DescribeTableInput{TableName: &TableName}

	result, err := svc.DescribeTable(tableInput)
	if err != nil {
		log.Fatalf("There was an error describing tables %v", err)
	}

	return result.Table.TableStatus

}

func TestShouldIncludeTable(t *testing.T) {
	mockTable := &dynamodb.TableDescription{
		TableName:        aws.String("cloud-nuke-test"),
		CreationDateTime: aws.Time(time.Now()),
	}

	mockExpression, err := regexp.Compile("^cloud-nuke-*")
	if err != nil {
		log.Fatalf("There was an error compiling regex expression %v", err)
	}

	mockExcludeConfig := config.Config{
		DynamoDB: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	mockIncludeConfig := config.Config{
		DynamoDB: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	cases := []struct {
		Name         string
		Table        *dynamodb.TableDescription
		Config       config.Config
		ExcludeAfter time.Time
		Expected     bool
	}{
		{
			Name:         "ConfigExclude",
			Table:        mockTable,
			Config:       mockExcludeConfig,
			ExcludeAfter: time.Now().Add(1 * time.Hour),
			Expected:     false,
		},
		{
			Name:         "ConfigInclude",
			Table:        mockTable,
			Config:       mockIncludeConfig,
			ExcludeAfter: time.Now().Add(1 * time.Hour),
			Expected:     true,
		},
		{
			Name:         "NotOlderThan",
			Table:        mockTable,
			Config:       config.Config{},
			ExcludeAfter: time.Now().Add(1 * time.Hour * -1),
			Expected:     false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := shouldIncludeTable(c.Table, c.ExcludeAfter, c.Config)
			assert.Equal(t, c.Expected, result)
		})
	}
}

func TestGetTablesDynamo(t *testing.T) {
	t.Parallel()
	region, err := getRandomRegion()
	require.NoError(t, err)

	db := DynamoDB{}
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	require.NoError(t, err)

	_, err = getAllDynamoTables(awsSession, time.Now().Add(1*time.Hour*-1), config.Config{}, db)
	require.NoError(t, err)
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
	require.NoError(t, err)

	tableName := "cloud-nuke-test-" + util.UniqueID()
	defer nukeAllDynamoDBTables(awsSession, []*string{&tableName})
	createTestDynamoTables(t, tableName, region)
	COUNTER := 0
	for COUNTER <= 1 {
		tableStatus := getTableStatus(tableName, region)
		if *tableStatus == "ACTIVE" {
			COUNTER += 1
			log.Printf("Created a table: %v\n", tableName)
		} else {
			log.Printf("Table not ready yet: %v", tableName)
		}
	}
	nukeErr := nukeAllDynamoDBTables(awsSession, []*string{&tableName})
	require.NoError(t, nukeErr)

	time.Sleep(5 * time.Second)

	tables, err := getAllDynamoTables(awsSession, time.Now().Add(1*time.Hour*-1), config.Config{}, db)
	require.NoError(t, err)

	for _, table := range tables {
		if tableName == *table {
			assert.Fail(t, errors.WithStackTrace(err).Error())
		}
	}
}

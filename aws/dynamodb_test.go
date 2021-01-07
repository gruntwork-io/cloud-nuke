package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func CreateTestDynamoTables(t *testing.T, session *session.Session) error {
	tables := 0
	var tablesToCreate []string
	// Creates 50 tables with Unique IDs to test the limiters as well as the deletes
	if tables < 60 {
		tablesToCreate = append(tablesToCreate, "dynamo-nuke-test-" + random.UniqueId() )
		tables +=1
	}


	svc := dynamodb.New(session)

	for _, table := range tablesToCreate {
		input := &dynamodb.CreateTableInput{TableName: aws.String(table)}
		_, err := svc.CreateTable(input)
		require.NoError(t, err)
	}

	return nil
}

func NukeAllDynamoDBTablesTest(t *testing.T, session *session.Session) error {
	tables, err := getAllDynamoTables(session, time.Now().Add(1*time.Hour*-1))
	require.NoError(t, err)

	nukeErr := nukeAllDynamoDBTables(session, tables)
	require.NoError(t, nukeErr)
	return nil
}

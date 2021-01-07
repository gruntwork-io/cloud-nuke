package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/require"
	"testing"
)

func CreateTestDynamoTables(t *testing.T, session *session.Session) error {
	svc := dynamodb.New(session)
	tablestocreate := []string{"Table1", "Table2"}
	for _, table := range tablestocreate {
		input := &dynamodb.CreateTableInput{TableName: aws.String(table)}
		_, err := svc.CreateTable(input)
		require.NoError(t, err)
	}

	return nil
}

func NukeAllDynamoDBTablesTest(t *testing.T, session *session.Session) error {
	tables, err := getAllDynamoTables(session)
	require.NoError(t, err)

	nukeErr := nukeAllDynamoDBTables(session, tables)
	require.NoError(t, nukeErr)
	return nil
}

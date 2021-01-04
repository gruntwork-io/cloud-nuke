package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func CreateTestDynamoTables(t *testing.T, session *session.Session) error {
	svc := dynamodb.New(session)
	tablestocreate := []string{"Table1", "Table2"}
	for _, table := range tablestocreate {
		input := &dynamodb.CreateTableInput{TableName: aws.String(table)}
		_, err := svc.CreateTable(input)
		if err != nil {
			assert.Fail(t, errors.WithStackTrace(err).Error())
		}
	}

	return nil
}

func NukeAllDynamoDBTablesTest(t *testing.T, session *session.Session) error {
	tables, err := getAllDynamoTables(session)
	t.Log(&tables)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	error := nukeAllDynamoDBTables(session, tables)
	if error != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	return nil
}

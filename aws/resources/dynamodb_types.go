package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type DynamoDB struct {
	Client           dynamodbiface.DynamoDBAPI
	Region           string
	DynamoTableNames []string
}

func (ddb *DynamoDB) Init(session *session.Session) {
	ddb.Client = dynamodb.New(session)
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

func (ddb *DynamoDB) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ddb.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ddb.DynamoTableNames = awsgo.StringValueSlice(identifiers)
	return ddb.DynamoTableNames, nil
}

// Nuke - nuke all Dynamo DB Tables
func (ddb *DynamoDB) Nuke(identifiers []string) error {
	if err := ddb.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

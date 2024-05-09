package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// SqsQueue - represents all sqs queues
type SqsQueue struct {
	BaseAwsResource
	Client    sqsiface.SQSAPI
	Region    string
	QueueUrls []string
}

func (sq *SqsQueue) Init(session *session.Session) {
	sq.Client = sqs.New(session)
}

// ResourceName - the simple name of the aws resource
func (sq *SqsQueue) ResourceName() string {
	return "sqs"
}

func (sq *SqsQueue) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The arns of the sqs queues
func (sq *SqsQueue) ResourceIdentifiers() []string {
	return sq.QueueUrls
}

func (sq *SqsQueue) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SQS
}

func (sq *SqsQueue) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sq.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sq.QueueUrls = awsgo.StringValueSlice(identifiers)
	return sq.QueueUrls, nil
}

// Nuke - nuke 'em all!!!
func (sq *SqsQueue) Nuke(identifiers []string) error {
	if err := sq.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// SqsQueue - represents all sqs queues
type SqsQueue struct {
	Client    sqsiface.SQSAPI
	Region    string
	QueueUrls []string
}

// ResourceName - the simple name of the aws resource
func (sq SqsQueue) ResourceName() string {
	return "sqs"
}

func (sq SqsQueue) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The arns of the sqs queues
func (sq SqsQueue) ResourceIdentifiers() []string {
	return sq.QueueUrls
}

// Nuke - nuke 'em all!!!
func (sq SqsQueue) Nuke(session *session.Session, identifiers []string) error {
	if err := sq.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

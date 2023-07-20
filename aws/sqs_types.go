package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// SQS - represents all sqs queues
type SQS struct {
	Client    sqsiface.SQSAPI
	Region    string
	QueueUrls []string
}

// ResourceName - the simple name of the aws resource
func (queue SQS) ResourceName() string {
	return "sqs"
}

func (queue SQS) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The arns of the sqs queues
func (queue SQS) ResourceIdentifiers() []string {
	return queue.QueueUrls
}

// Nuke - nuke 'em all!!!
func (queue SQS) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllSqsQueues(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

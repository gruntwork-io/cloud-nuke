package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// SqsQueue - represents all sqs queues
type SqsQueue struct {
	QueueUrls []string
}

// ResourceName - the simple name of the aws resource
func (queue SqsQueue) ResourceName() string {
	return "sqs"
}

func (queue SqsQueue) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The arns of the sqs queues
func (queue SqsQueue) ResourceIdentifiers() []string {
	return queue.QueueUrls
}

// Nuke - nuke 'em all!!!
func (queue SqsQueue) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllSqsQueues(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

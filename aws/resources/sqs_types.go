package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SqsQueueAPI interface {
	DeleteQueue(ctx context.Context, params *sqs.DeleteQueueInput, optFns ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	ListQueues(context.Context, *sqs.ListQueuesInput, ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error)
}

// SqsQueue - represents all sqs queues
type SqsQueue struct {
	BaseAwsResource
	Client    SqsQueueAPI
	Region    string
	QueueUrls []string
}

func (sq *SqsQueue) Init(cfg aws.Config) {
	sq.Client = sqs.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (sq *SqsQueue) ResourceName() string {
	return "sqs"
}

func (sq *SqsQueue) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The arn's of the sqs queues
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

	sq.QueueUrls = aws.ToStringSlice(identifiers)
	return sq.QueueUrls, nil
}

// Nuke - nuke 'em all!!!
func (sq *SqsQueue) Nuke(ctx context.Context, identifiers []string) error {
	if err := sq.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

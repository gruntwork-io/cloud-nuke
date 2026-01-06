package resources

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// SqsQueueAPI defines the interface for SQS Queue operations.
type SqsQueueAPI interface {
	ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	DeleteQueue(ctx context.Context, params *sqs.DeleteQueueInput, optFns ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error)
}

// NewSqsQueue creates a new SqsQueue resource using the generic resource pattern.
func NewSqsQueue() AwsResource {
	return NewAwsResource(&resource.Resource[SqsQueueAPI]{
		ResourceTypeName: "sqs",
		BatchSize:        DefaultBatchSize, // Tentative batch size to ensure AWS doesn't throttle
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SqsQueueAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = sqs.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SQS
		},
		Lister: listSqsQueues,
		Nuker:  resource.SimpleBatchDeleter(deleteSqsQueue),
	})
}

// listSqsQueues retrieves all SQS Queues that match the config filters.
func listSqsQueues(ctx context.Context, client SqsQueueAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var queueUrls []*string

	paginator := sqs.NewListQueuesPaginator(client, &sqs.ListQueuesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, queueUrl := range page.QueueUrls {
			createdAt, err := getQueueCreatedTime(ctx, client, queueUrl)
			if err != nil {
				return nil, err
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: aws.String(queueUrl),
				Time: createdAt,
			}) {
				queueUrls = append(queueUrls, aws.String(queueUrl))
			}
		}
	}

	return queueUrls, nil
}

// getQueueCreatedTime retrieves the creation timestamp for an SQS queue.
func getQueueCreatedTime(ctx context.Context, client SqsQueueAPI, queueUrl string) (*time.Time, error) {
	attributes, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueUrl),
		AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameCreatedTimestamp},
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	createdAt, ok := attributes.Attributes["CreatedTimestamp"]
	if !ok {
		return nil, errors.WithStackTrace(fmt.Errorf("expected to find CreatedTimestamp attribute for queue %s", queueUrl))
	}

	createdAtInt, err := strconv.ParseInt(createdAt, 10, 64)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	createdAtTime := time.Unix(createdAtInt, 0)
	return &createdAtTime, nil
}

// deleteSqsQueue deletes a single SQS Queue.
func deleteSqsQueue(ctx context.Context, client SqsQueueAPI, url *string) error {
	_, err := client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: url,
	})
	return errors.WithStackTrace(err)
}

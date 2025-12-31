package resources

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// SqsQueueAPI defines the interface for SQS Queue operations.
type SqsQueueAPI interface {
	DeleteQueue(ctx context.Context, params *sqs.DeleteQueueInput, optFns ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error)
}

// NewSqsQueue creates a new SQS Queue resource using the generic resource pattern.
func NewSqsQueue() AwsResource {
	return NewAwsResource(&resource.Resource[SqsQueueAPI]{
		ResourceTypeName: "sqs",
		BatchSize:        49,
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

// listSqsQueues retrieves all SQS queues that match the config filters.
func listSqsQueues(ctx context.Context, client SqsQueueAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var queueUrls []*string

	paginator := sqs.NewListQueuesPaginator(client, &sqs.ListQueuesInput{
		MaxResults: aws.Int32(10),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, queueUrl := range page.QueueUrls {
			queueAttributes, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
				QueueUrl:       aws.String(queueUrl),
				AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameCreatedTimestamp},
			})
			if err != nil {
				return nil, err
			}

			createdAt, ok := queueAttributes.Attributes["CreatedTimestamp"]
			if !ok {
				continue
			}

			createdAtInt, err := strconv.ParseInt(createdAt, 10, 64)
			if err != nil {
				continue
			}
			createdAtTime := time.Unix(createdAtInt, 0)

			if cfg.ShouldInclude(config.ResourceValue{
				Name: aws.String(queueUrl),
				Time: &createdAtTime,
			}) {
				queueUrls = append(queueUrls, aws.String(queueUrl))
			}
		}
	}

	return queueUrls, nil
}

// deleteSqsQueue deletes a single SQS queue.
func deleteSqsQueue(ctx context.Context, client SqsQueueAPI, queueUrl *string) error {
	_, err := client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: queueUrl,
	})
	return err
}

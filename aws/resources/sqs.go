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
)

// SqsQueueAPI defines the interface for SQS Queue operations.
type SqsQueueAPI interface {
	DeleteQueue(ctx context.Context, params *sqs.DeleteQueueInput, optFns ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error)
}

// NewSqsQueue creates a new SqsQueue resource using the generic resource pattern.
func NewSqsQueue() AwsResource {
	return NewAwsResource(&resource.Resource[SqsQueueAPI]{
		ResourceTypeName: "sqs",
		BatchSize:        49, // Tentative batch size to ensure AWS doesn't throttle
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
	var allQueueUrls []*string

	paginator := sqs.NewListQueuesPaginator(client, &sqs.ListQueuesInput{
		MaxResults: aws.Int32(10),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		allQueueUrls = append(allQueueUrls, aws.StringSlice(page.QueueUrls)...)
	}

	var filteredUrls []*string
	for _, queueUrl := range allQueueUrls {
		attributes, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl:       queueUrl,
			AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameCreatedTimestamp},
		})
		if err != nil {
			return nil, err
		}

		createdAt, ok := attributes.Attributes["CreatedTimestamp"]
		if !ok {
			return nil, fmt.Errorf("expected to find CreatedTimestamp attribute")
		}

		createdAtInt, err := strconv.ParseInt(createdAt, 10, 64)
		if err != nil {
			return nil, err
		}
		createdAtTime := time.Unix(createdAtInt, 0)

		if cfg.ShouldInclude(config.ResourceValue{
			Name: queueUrl,
			Time: &createdAtTime,
		}) {
			filteredUrls = append(filteredUrls, queueUrl)
		}
	}

	return filteredUrls, nil
}

// deleteSqsQueue deletes a single SQS Queue.
func deleteSqsQueue(ctx context.Context, client SqsQueueAPI, url *string) error {
	_, err := client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: url,
	})
	return err
}

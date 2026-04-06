package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	"github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// MQBrokerAPI defines the interface for Amazon MQ broker operations.
type MQBrokerAPI interface {
	ListBrokers(ctx context.Context, params *mq.ListBrokersInput, optFns ...func(*mq.Options)) (*mq.ListBrokersOutput, error)
	ListTags(ctx context.Context, params *mq.ListTagsInput, optFns ...func(*mq.Options)) (*mq.ListTagsOutput, error)
	DeleteBroker(ctx context.Context, params *mq.DeleteBrokerInput, optFns ...func(*mq.Options)) (*mq.DeleteBrokerOutput, error)
}

// NewMQBroker creates a new Amazon MQ broker resource.
func NewMQBroker() AwsResource {
	return NewAwsResource(&resource.Resource[MQBrokerAPI]{
		ResourceTypeName: "mq-broker",
		BatchSize:        20,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[MQBrokerAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = mq.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.MQBroker
		},
		Lister: listMQBrokers,
		Nuker:  resource.SimpleBatchDeleter(deleteMQBroker),
	})
}

// listMQBrokers retrieves all Amazon MQ brokers that match the config filters.
func listMQBrokers(ctx context.Context, client MQBrokerAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string
	paginator := mq.NewListBrokersPaginator(client, &mq.ListBrokersInput{
		MaxResults: aws.Int32(100),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, broker := range page.BrokerSummaries {
			// Skip brokers in non-deletable states
			switch broker.BrokerState {
			case types.BrokerStateCreationInProgress, types.BrokerStateDeletionInProgress, types.BrokerStateCreationFailed:
				continue
			}

			var tags map[string]string
			tagsOutput, err := client.ListTags(ctx, &mq.ListTagsInput{
				ResourceArn: broker.BrokerArn,
			})
			if err != nil {
				logging.Debugf("Error listing tags for MQ broker %s: %v", aws.ToString(broker.BrokerName), err)
			} else if tagsOutput != nil {
				tags = tagsOutput.Tags
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: broker.BrokerName,
				Time: broker.Created,
				Tags: tags,
			}) {
				identifiers = append(identifiers, broker.BrokerId)
			}
		}
	}

	return identifiers, nil
}

// deleteMQBroker deletes a single Amazon MQ broker.
func deleteMQBroker(ctx context.Context, client MQBrokerAPI, brokerId *string) error {
	_, err := client.DeleteBroker(ctx, &mq.DeleteBrokerInput{
		BrokerId: brokerId,
	})
	return err
}

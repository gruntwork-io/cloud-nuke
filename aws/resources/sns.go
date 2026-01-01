package resources

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// SNSTopicAPI defines the interface for SNS Topic operations.
type SNSTopicAPI interface {
	DeleteTopic(ctx context.Context, params *sns.DeleteTopicInput, optFns ...func(*sns.Options)) (*sns.DeleteTopicOutput, error)
	ListTopics(ctx context.Context, params *sns.ListTopicsInput, optFns ...func(*sns.Options)) (*sns.ListTopicsOutput, error)
	ListTagsForResource(ctx context.Context, params *sns.ListTagsForResourceInput, optFns ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error)
	TagResource(ctx context.Context, params *sns.TagResourceInput, optFns ...func(*sns.Options)) (*sns.TagResourceOutput, error)
}

// NewSNSTopic creates a new SNSTopic resource using the generic resource pattern.
func NewSNSTopic() AwsResource {
	return NewAwsResource(&resource.Resource[SNSTopicAPI]{
		ResourceTypeName: "snstopic",
		BatchSize:        50,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SNSTopicAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = sns.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SNS
		},
		Lister: listSNSTopics,
		Nuker:  resource.SimpleBatchDeleter(deleteSNSTopic),
	})
}

// listSNSTopics retrieves all SNS topics that match the config filters.
// SNS APIs do not return a creation date, so we tag resources with a first-seen time
// when the topic first appears, then use that tag for time-based filtering.
func listSNSTopics(ctx context.Context, client SNSTopicAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var snsTopics []*string

	excludeFirstSeenTag, err := util.GetBoolFromContext(ctx, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, err
	}

	paginator := sns.NewListTopicsPaginator(client, &sns.ListTopicsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, topic := range page.Topics {
			var firstSeenTime *time.Time

			if !excludeFirstSeenTag {
				firstSeenTime, err = getFirstSeenSNSTag(ctx, client, *topic.TopicArn)
				if err != nil {
					logging.Errorf("Unable to retrieve tags for SNS Topic: %s, with error: %s", *topic.TopicArn, err)
					continue
				}

				if firstSeenTime == nil {
					now := time.Now().UTC()
					firstSeenTime = &now
					if err := setFirstSeenSNSTag(ctx, client, *topic.TopicArn, now); err != nil {
						logging.Errorf("Unable to apply first seen tag to SNS Topic: %s, with error: %s", *topic.TopicArn, err)
						continue
					}
				}
			}

			// Extract topic name from ARN (format: arn:aws:sns:us-east-1:123456789012:MyTopic)
			nameIndex := strings.LastIndex(*topic.TopicArn, ":")
			topicName := (*topic.TopicArn)[nameIndex+1:]

			if cfg.ShouldInclude(config.ResourceValue{
				Time: firstSeenTime,
				Name: &topicName,
			}) {
				snsTopics = append(snsTopics, topic.TopicArn)
			}
		}
	}

	return snsTopics, nil
}

// getFirstSeenSNSTag retrieves the first-seen time tag for an SNS topic.
func getFirstSeenSNSTag(ctx context.Context, client SNSTopicAPI, topicArn string) (*time.Time, error) {
	response, err := client.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{
		ResourceArn: &topicArn,
	})
	if err != nil {
		return nil, err
	}

	for _, tag := range response.Tags {
		if util.IsFirstSeenTag(tag.Key) {
			return util.ParseTimestamp(tag.Value)
		}
	}

	return nil, nil
}

// setFirstSeenSNSTag sets the first-seen time tag on an SNS topic.
func setFirstSeenSNSTag(ctx context.Context, client SNSTopicAPI, topicArn string, value time.Time) error {
	_, err := client.TagResource(ctx, &sns.TagResourceInput{
		ResourceArn: &topicArn,
		Tags: []types.Tag{
			{
				Key:   aws.String(util.FirstSeenTagKey),
				Value: aws.String(util.FormatTimestamp(value)),
			},
		},
	})
	return err
}

// deleteSNSTopic deletes a single SNS topic.
func deleteSNSTopic(ctx context.Context, client SNSTopicAPI, topicArn *string) error {
	_, err := client.DeleteTopic(ctx, &sns.DeleteTopicInput{
		TopicArn: topicArn,
	})
	return err
}

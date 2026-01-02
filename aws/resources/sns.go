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
	"github.com/gruntwork-io/go-commons/errors"
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
		BatchSize:        DefaultBatchSize,
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
	excludeFirstSeenTag, err := util.GetBoolFromContext(ctx, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var topics []*string
	paginator := sns.NewListTopicsPaginator(client, &sns.ListTopicsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, topic := range page.Topics {
			firstSeenTime, err := getFirstSeenSNSTagOrSet(ctx, client, *topic.TopicArn, excludeFirstSeenTag)
			if err != nil {
				logging.Errorf("Unable to process first-seen tag for SNS Topic %s: %s", *topic.TopicArn, err)
				continue
			}

			// Extract topic name from ARN (format: arn:aws:sns:us-east-1:123456789012:MyTopic)
			topicName := (*topic.TopicArn)[strings.LastIndex(*topic.TopicArn, ":")+1:]

			if cfg.ShouldInclude(config.ResourceValue{
				Time: firstSeenTime,
				Name: &topicName,
			}) {
				topics = append(topics, topic.TopicArn)
			}
		}
	}

	return topics, nil
}

// getFirstSeenSNSTagOrSet retrieves the first-seen timestamp or sets it if not present.
// If excludeFirstSeenTag is true, skips tag processing and returns nil.
func getFirstSeenSNSTagOrSet(ctx context.Context, client SNSTopicAPI, topicArn string, excludeFirstSeenTag bool) (*time.Time, error) {
	if excludeFirstSeenTag {
		return nil, nil
	}

	firstSeenTime, err := getFirstSeenSNSTag(ctx, client, topicArn)
	if err != nil {
		return nil, err
	}

	if firstSeenTime == nil {
		now := time.Now().UTC()
		if err := setFirstSeenSNSTag(ctx, client, topicArn, now); err != nil {
			return nil, err
		}
		return &now, nil
	}

	return firstSeenTime, nil
}

// getFirstSeenSNSTag retrieves the first-seen time tag for an SNS topic.
func getFirstSeenSNSTag(ctx context.Context, client SNSTopicAPI, topicArn string) (*time.Time, error) {
	response, err := client.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{
		ResourceArn: &topicArn,
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
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
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

// deleteSNSTopic deletes a single SNS topic.
func deleteSNSTopic(ctx context.Context, client SNSTopicAPI, topicArn *string) error {
	_, err := client.DeleteTopic(ctx, &sns.DeleteTopicInput{
		TopicArn: topicArn,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

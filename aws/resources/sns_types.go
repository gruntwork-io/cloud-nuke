package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SNSTopicAPI interface {
	DeleteTopic(ctx context.Context, params *sns.DeleteTopicInput, optFns ...func(*sns.Options)) (*sns.DeleteTopicOutput, error)
	ListTopics(context.Context, *sns.ListTopicsInput, ...func(*sns.Options)) (*sns.ListTopicsOutput, error)
	ListTagsForResource(ctx context.Context, params *sns.ListTagsForResourceInput, optFns ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error)
	TagResource(ctx context.Context, params *sns.TagResourceInput, optFns ...func(*sns.Options)) (*sns.TagResourceOutput, error)
}

type SNSTopic struct {
	BaseAwsResource
	Client SNSTopicAPI
	Region string
	Arns   []string
}

func (s *SNSTopic) InitV2(cfg aws.Config) {
	s.Client = sns.NewFromConfig(cfg)
}

func (s *SNSTopic) ResourceName() string {
	return "snstopic"
}

func (s *SNSTopic) ResourceIdentifiers() []string {
	return s.Arns
}

func (s *SNSTopic) MaxBatchSize() int {
	return 50
}

func (s *SNSTopic) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SNS
}

func (s *SNSTopic) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := s.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	s.Arns = aws.ToStringSlice(identifiers)
	return s.Arns, nil
}

func (s *SNSTopic) Nuke(identifiers []string) error {
	if err := s.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

// custom errors

type TooManySNSTopicsErr struct{}

func (err TooManySNSTopicsErr) Error() string {
	return "Too many SNS Topics requested at once."
}

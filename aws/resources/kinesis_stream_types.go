package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type KinesisStreamsAPI interface {
	ListStreams(ctx context.Context, params *kinesis.ListStreamsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamsOutput, error)
	DeleteStream(ctx context.Context, params *kinesis.DeleteStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DeleteStreamOutput, error)
}

// KinesisStreams - represents all Kinesis streams
type KinesisStreams struct {
	BaseAwsResource
	Client KinesisStreamsAPI
	Region string
	Names  []string
}

func (ks *KinesisStreams) InitV2(cfg aws.Config) {
	ks.Client = kinesis.NewFromConfig(cfg)
}

// ResourceName - The simple name of the AWS resource
func (ks *KinesisStreams) ResourceName() string {
	return "kinesis-stream"
}

// ResourceIdentifiers - The names of the Kinesis Streams
func (ks *KinesisStreams) ResourceIdentifiers() []string {
	return ks.Names
}

func (ks *KinesisStreams) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that Kinesis Streams does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We pick 35 here, which is half of what the AWS web
	// console will do. We pick a conservative number here to avoid hitting AWS API rate limits.
	return 35
}

func (ks *KinesisStreams) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.KinesisStream
}

func (ks *KinesisStreams) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ks.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ks.Names = aws.ToStringSlice(identifiers)
	return ks.Names, nil
}

// Nuke - nuke 'em all!!!
func (ks *KinesisStreams) Nuke(identifiers []string) error {
	if err := ks.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

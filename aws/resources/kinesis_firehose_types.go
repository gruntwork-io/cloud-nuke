package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type KinesisFirehose struct {
	BaseAwsResource
	Client firehoseiface.FirehoseAPI
	Region string
	Names  []string
}

func (kf *KinesisFirehose) Init(session *session.Session) {
	kf.Client = firehose.New(session)
}

// ResourceName - The simple name of the AWS resource
func (kf *KinesisFirehose) ResourceName() string {
	return "kinesis-firehose"
}

// ResourceIdentifiers - The names of the Kinesis Streams
func (kf *KinesisFirehose) ResourceIdentifiers() []string {
	return kf.Names
}

func (kf *KinesisFirehose) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that Kinesis Streams does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We pick 35 here, which is half of what the AWS web
	// console will do. We pick a conservative number here to avoid hitting AWS API rate limits.
	return 35
}

func (kf *KinesisFirehose) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.KinesisFirehose
}

func (kf *KinesisFirehose) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := kf.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	kf.Names = aws.StringValueSlice(identifiers)
	return kf.Names, nil
}

// Nuke - nuke 'em all!!!
func (kf *KinesisFirehose) Nuke(identifiers []string) error {
	if err := kf.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

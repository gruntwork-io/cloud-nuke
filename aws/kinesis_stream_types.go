package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// KinesisStreams - represents all Kinesis streams
type KinesisStreams struct {
	Client kinesisiface.KinesisAPI
	Region string
	Names  []string
}

// ResourceName - The simple name of the AWS resource
func (ks KinesisStreams) ResourceName() string {
	return "kinesis-stream"
}

// ResourceIdentifiers - The names of the Kinesis Streams
func (ks KinesisStreams) ResourceIdentifiers() []string {
	return ks.Names
}

func (ks KinesisStreams) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that Kinesis Streams does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We pick 35 here, which is half of what the AWS web
	// console will do. We pick a conservative number here to avoid hitting AWS API rate limits.
	return 35
}

func (ks KinesisStreams) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := ks.getAll(configObj)
	if err != nil {
		return nil, err
	}

	ks.Names = aws.StringValueSlice(identifiers)
	return ks.Names, nil
}

// Nuke - nuke 'em all!!!
func (ks KinesisStreams) Nuke(identifiers []string) error {
	if err := ks.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

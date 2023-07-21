package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// KinesisStreams - represents all Kinesis streams
type KinesisStreams struct {
	Client kinesisiface.KinesisAPI
	Region string
	Names  []string
}

// ResourceName - The simple name of the AWS resource
func (k KinesisStreams) ResourceName() string {
	return "kinesis-stream"
}

// ResourceIdentifiers - The names of the Kinesis Streams
func (k KinesisStreams) ResourceIdentifiers() []string {
	return k.Names
}

func (k KinesisStreams) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that Kinesis Streams does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We pick 35 here, which is half of what the AWS web
	// console will do. We pick a conservative number here to avoid hitting AWS API rate limits.
	return 35
}

// Nuke - nuke 'em all!!!
func (k KinesisStreams) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllKinesisStreams(session, aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

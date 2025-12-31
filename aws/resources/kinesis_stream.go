package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// KinesisStreamsAPI defines the interface for Kinesis Streams operations.
type KinesisStreamsAPI interface {
	ListStreams(ctx context.Context, params *kinesis.ListStreamsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamsOutput, error)
	DeleteStream(ctx context.Context, params *kinesis.DeleteStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DeleteStreamOutput, error)
}

// NewKinesisStreams creates a new Kinesis Streams resource using the generic resource pattern.
func NewKinesisStreams() AwsResource {
	return NewAwsResource(&resource.Resource[KinesisStreamsAPI]{
		ResourceTypeName: "kinesis-stream",
		BatchSize:        35, // Conservative batch size to avoid hitting AWS API rate limits
		InitClient: WrapAwsInitClient(func(r *resource.Resource[KinesisStreamsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = kinesis.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.KinesisStream
		},
		Lister: listKinesisStreams,
		Nuker:  resource.SimpleBatchDeleter(deleteKinesisStream),
	})
}

// listKinesisStreams retrieves all Kinesis streams that match the config filters.
func listKinesisStreams(ctx context.Context, client KinesisStreamsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allStreams []*string

	paginator := kinesis.NewListStreamsPaginator(client, &kinesis.ListStreamsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, stream := range page.StreamNames {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: aws.String(stream),
			}) {
				allStreams = append(allStreams, aws.String(stream))
			}
		}
	}

	return allStreams, nil
}

// deleteKinesisStream deletes a single Kinesis stream.
func deleteKinesisStream(ctx context.Context, client KinesisStreamsAPI, streamName *string) error {
	_, err := client.DeleteStream(ctx, &kinesis.DeleteStreamInput{
		StreamName: streamName,
	})
	return err
}

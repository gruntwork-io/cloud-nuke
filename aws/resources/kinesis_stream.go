package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// NewKinesisStreams creates a new Kinesis Streams resource using the generic resource pattern.
func NewKinesisStreams() AwsResource {
	return NewAwsResource(&resource.Resource[*kinesis.Client]{
		ResourceTypeName: "kinesis-stream",
		BatchSize:        35, // Conservative batch size to avoid hitting AWS API rate limits
		InitClient: func(r *resource.Resource[*kinesis.Client], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for Kinesis client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = kinesis.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.KinesisStream
		},
		Lister: listKinesisStreams,
		Nuker:  resource.SimpleBatchDeleter(deleteKinesisStream),
	})
}

// listKinesisStreams retrieves all Kinesis streams that match the config filters.
func listKinesisStreams(ctx context.Context, client *kinesis.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
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
func deleteKinesisStream(ctx context.Context, client *kinesis.Client, streamName *string) error {
	_, err := client.DeleteStream(ctx, &kinesis.DeleteStreamInput{
		StreamName: streamName,
	})
	return err
}

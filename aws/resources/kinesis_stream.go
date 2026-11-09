package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// KinesisStreamsAPI defines the interface for Kinesis Streams operations.
type KinesisStreamsAPI interface {
	ListStreams(ctx context.Context, params *kinesis.ListStreamsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamsOutput, error)
	DescribeStreamSummary(ctx context.Context, params *kinesis.DescribeStreamSummaryInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeStreamSummaryOutput, error)
	DeleteStream(ctx context.Context, params *kinesis.DeleteStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DeleteStreamOutput, error)
	ListTagsForStream(ctx context.Context, params *kinesis.ListTagsForStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.ListTagsForStreamOutput, error)
}

// NewKinesisStreams creates a new Kinesis Streams resource using the generic resource pattern.
func NewKinesisStreams() AwsResource {
	return NewAwsResource(&resource.Resource[KinesisStreamsAPI]{
		ResourceTypeName: "kinesis-stream",
		// Conservative batch size to avoid AWS API rate limiting for sequential delete calls.
		BatchSize: 35,
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

		createdAt := make(map[string]*time.Time, len(page.StreamSummaries))
		for _, summary := range page.StreamSummaries {
			createdAt[aws.ToString(summary.StreamName)] = summary.StreamCreationTimestamp
		}

		for _, stream := range page.StreamNames {
			tagsOutput, err := client.ListTagsForStream(ctx, &kinesis.ListTagsForStreamInput{
				StreamName: aws.String(stream),
			})
			if err != nil {
				logging.Debugf("[Failed] Unable to fetch tags for Kinesis stream %s: %s", stream, err)
				continue
			}

			streamCreatedAt := createdAt[stream]
			if streamCreatedAt == nil && cfg.HasTimeFilter() {
				streamCreatedAt = describeKinesisStreamCreationTime(ctx, client, stream)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: aws.String(stream),
				Time: streamCreatedAt,
				Tags: util.ConvertKinesisTagsToMap(tagsOutput.Tags),
			}) {
				allStreams = append(allStreams, aws.String(stream))
			}
		}
	}

	return allStreams, nil
}

// describeKinesisStreamCreationTime reads the creation time of a stream that
// ListStreams returned without a summary. It returns nil, and warns, when the
// time cannot be read, since a time filter then cannot include the stream.
func describeKinesisStreamCreationTime(ctx context.Context, client KinesisStreamsAPI, stream string) *time.Time {
	output, err := client.DescribeStreamSummary(ctx, &kinesis.DescribeStreamSummaryInput{
		StreamName: aws.String(stream),
	})
	if err != nil || output.StreamDescriptionSummary == nil {
		logging.Warnf("Unable to read the creation time of Kinesis stream %s, so a time filter cannot include it: %v", stream, err)
		return nil
	}
	return output.StreamDescriptionSummary.StreamCreationTimestamp
}

// deleteKinesisStream deletes a single Kinesis stream.
// EnforceConsumerDeletion is set to true to delete streams even if they have registered consumers.
func deleteKinesisStream(ctx context.Context, client KinesisStreamsAPI, streamName *string) error {
	_, err := client.DeleteStream(ctx, &kinesis.DeleteStreamInput{
		StreamName:              streamName,
		EnforceConsumerDeletion: aws.Bool(true),
	})
	return err
}

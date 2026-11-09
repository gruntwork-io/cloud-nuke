package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// KinesisFirehoseAPI defines the interface for Kinesis Firehose operations.
type KinesisFirehoseAPI interface {
	ListDeliveryStreams(ctx context.Context, params *firehose.ListDeliveryStreamsInput, optFns ...func(*firehose.Options)) (*firehose.ListDeliveryStreamsOutput, error)
	DeleteDeliveryStream(ctx context.Context, params *firehose.DeleteDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DeleteDeliveryStreamOutput, error)
	DescribeDeliveryStream(ctx context.Context, params *firehose.DescribeDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DescribeDeliveryStreamOutput, error)
	ListTagsForDeliveryStream(ctx context.Context, params *firehose.ListTagsForDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.ListTagsForDeliveryStreamOutput, error)
}

// NewKinesisFirehose creates a new Kinesis Firehose resource using the generic resource pattern.
func NewKinesisFirehose() AwsResource {
	return NewAwsResource(&resource.Resource[KinesisFirehoseAPI]{
		ResourceTypeName: "kinesis-firehose",
		// Kinesis Firehose does not support bulk delete, so we delete in parallel using goroutines.
		// Using 35 (half of what the AWS console does) to avoid rate limiting.
		BatchSize: 35,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[KinesisFirehoseAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = firehose.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.KinesisFirehose
		},
		Lister: listKinesisFirehose,
		Nuker:  resource.SimpleBatchDeleter(deleteKinesisFirehose),
	})
}

// listKinesisFirehose retrieves all Kinesis Firehose delivery streams that match the config filters.
func listKinesisFirehose(ctx context.Context, client KinesisFirehoseAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string
	var exclusiveStartName *string

	for {
		output, err := client.ListDeliveryStreams(ctx, &firehose.ListDeliveryStreamsInput{
			ExclusiveStartDeliveryStreamName: exclusiveStartName,
		})
		if err != nil {
			return nil, err
		}

		for _, stream := range output.DeliveryStreamNames {
			tagsOutput, err := client.ListTagsForDeliveryStream(ctx, &firehose.ListTagsForDeliveryStreamInput{
				DeliveryStreamName: aws.String(stream),
			})
			if err != nil {
				logging.Debugf("[Failed] Unable to fetch tags for Kinesis Firehose stream %s: %s", stream, err)
				continue
			}

			var createdAt *time.Time
			if cfg.HasTimeFilter() {
				createdAt = describeKinesisFirehoseCreationTime(ctx, client, stream)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: aws.String(stream),
				Time: createdAt,
				Tags: util.ConvertFirehoseTagsToMap(tagsOutput.Tags),
			}) {
				ids = append(ids, aws.String(stream))
			}
		}

		if !aws.ToBool(output.HasMoreDeliveryStreams) || len(output.DeliveryStreamNames) == 0 {
			break
		}
		exclusiveStartName = aws.String(output.DeliveryStreamNames[len(output.DeliveryStreamNames)-1])
	}

	return ids, nil
}

// describeKinesisFirehoseCreationTime reads the creation time of a delivery
// stream, which ListDeliveryStreams does not return. It returns nil, and warns,
// when the time cannot be read, since a time filter then cannot include the stream.
func describeKinesisFirehoseCreationTime(ctx context.Context, client KinesisFirehoseAPI, stream string) *time.Time {
	output, err := client.DescribeDeliveryStream(ctx, &firehose.DescribeDeliveryStreamInput{
		DeliveryStreamName: aws.String(stream),
	})
	if err != nil || output.DeliveryStreamDescription == nil {
		logging.Warnf("Unable to read the creation time of Kinesis Firehose stream %s, so a time filter cannot include it: %v", stream, err)
		return nil
	}
	return output.DeliveryStreamDescription.CreateTimestamp
}

// deleteKinesisFirehose deletes a single Kinesis Firehose delivery stream.
func deleteKinesisFirehose(ctx context.Context, client KinesisFirehoseAPI, id *string) error {
	_, err := client.DeleteDeliveryStream(ctx, &firehose.DeleteDeliveryStreamInput{
		AllowForceDelete:   aws.Bool(true),
		DeliveryStreamName: id,
	})
	return err
}

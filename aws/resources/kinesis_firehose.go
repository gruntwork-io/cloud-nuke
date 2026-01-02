package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// KinesisFirehoseAPI defines the interface for Kinesis Firehose operations.
type KinesisFirehoseAPI interface {
	ListDeliveryStreams(ctx context.Context, params *firehose.ListDeliveryStreamsInput, optFns ...func(*firehose.Options)) (*firehose.ListDeliveryStreamsOutput, error)
	DeleteDeliveryStream(ctx context.Context, params *firehose.DeleteDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DeleteDeliveryStreamOutput, error)
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
			if cfg.ShouldInclude(config.ResourceValue{
				Name: aws.String(stream),
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

// deleteKinesisFirehose deletes a single Kinesis Firehose delivery stream.
func deleteKinesisFirehose(ctx context.Context, client KinesisFirehoseAPI, id *string) error {
	_, err := client.DeleteDeliveryStream(ctx, &firehose.DeleteDeliveryStreamInput{
		AllowForceDelete:   aws.Bool(true),
		DeliveryStreamName: id,
	})
	return err
}

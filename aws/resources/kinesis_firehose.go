package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// NewKinesisFirehose creates a new Kinesis Firehose resource using the generic resource pattern.
func NewKinesisFirehose() AwsResource {
	return NewAwsResource(&resource.Resource[*firehose.Client]{
		ResourceTypeName: "kinesis-firehose",
		// Tentative batch size to ensure AWS doesn't throttle. Note that Kinesis Firehose does not support bulk delete,
		// so we will be deleting this many in parallel using go routines. We pick 35 here, which is half of what the
		// AWS web console will do. We pick a conservative number here to avoid hitting AWS API rate limits.
		BatchSize: 35,
		InitClient: func(r *resource.Resource[*firehose.Client], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for Firehose client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = firehose.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.KinesisFirehose
		},
		Lister: listKinesisFirehose,
		Nuker:  resource.SimpleBatchDeleter(deleteKinesisFirehose),
	})
}

// listKinesisFirehose retrieves all Kinesis Firehose delivery streams that match the config filters.
func listKinesisFirehose(ctx context.Context, client *firehose.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.ListDeliveryStreams(ctx, &firehose.ListDeliveryStreamsInput{})
	if err != nil {
		return nil, err
	}

	var ids []*string
	for _, stream := range output.DeliveryStreamNames {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: aws.String(stream),
		}) {
			ids = append(ids, aws.String(stream))
		}
	}

	return ids, nil
}

// deleteKinesisFirehose deletes a single Kinesis Firehose delivery stream.
func deleteKinesisFirehose(ctx context.Context, client *firehose.Client, id *string) error {
	_, err := client.DeleteDeliveryStream(ctx, &firehose.DeleteDeliveryStreamInput{
		AllowForceDelete:   aws.Bool(true),
		DeliveryStreamName: id,
	})
	if err != nil {
		return err
	}

	logging.Debugf("Deleted Kinesis Firehose: %s", aws.ToString(id))
	return nil
}

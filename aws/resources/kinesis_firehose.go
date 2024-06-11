package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (kf *KinesisFirehose) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	allStreams := []*string{}
	output, err := kf.Client.ListDeliveryStreamsWithContext(kf.Context, &firehose.ListDeliveryStreamsInput{})

	for _, stream := range output.DeliveryStreamNames {
		if configObj.KinesisFirehose.ShouldInclude(config.ResourceValue{
			Name: stream,
		}) {
			allStreams = append(allStreams, stream)
		}
	}

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return allStreams, nil
}

func (kf *KinesisFirehose) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Kinesis Firhose to nuke in region %s", kf.Region)
		return nil
	}

	logging.Debugf("Deleting all Kinesis Firhose in region %s", kf.Region)
	var deleted []*string
	for _, id := range identifiers {
		_, err := kf.Client.DeleteDeliveryStreamWithContext(kf.Context, &firehose.DeleteDeliveryStreamInput{
			AllowForceDelete:   awsgo.Bool(true),
			DeliveryStreamName: id,
		})
		e := report.Entry{
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Kinesis Firehose",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deleted = append(deleted, id)
			logging.Debugf("Deleted Kinesis Firehose: %s", *id)
		}
	}

	logging.Debugf("[OK] %d Kinesis Firehose(s) deleted in %s", len(deleted), kf.Region)

	return nil
}

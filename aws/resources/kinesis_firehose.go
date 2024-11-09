package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (kf *KinesisFirehose) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var allStreams []*string
	output, err := kf.Client.ListDeliveryStreams(kf.Context, &firehose.ListDeliveryStreamsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, stream := range output.DeliveryStreamNames {
		if configObj.KinesisFirehose.ShouldInclude(config.ResourceValue{
			Name: aws.String(stream),
		}) {
			allStreams = append(allStreams, aws.String(stream))
		}
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
		_, err := kf.Client.DeleteDeliveryStream(kf.Context, &firehose.DeleteDeliveryStreamInput{
			AllowForceDelete:   aws.Bool(true),
			DeliveryStreamName: id,
		})
		e := report.Entry{
			Identifier:   aws.ToString(id),
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

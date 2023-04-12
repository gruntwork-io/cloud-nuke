package aws

import (
	"context"
	"sync"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllKinesisStreams(session *session.Session, configObj config.Config) ([]*string, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}
	svc := kinesis.NewFromConfig(cfg)

	allStreams := []*string{}

	paginator := kinesis.NewListStreamsPaginator(svc, nil)

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(context.TODO())
		if err != nil {
			return []*string{}, errors.WithStackTrace(err)
		}
		for _, stream := range resp.StreamNames {
			if shouldIncludeKinesisStream(aws.String(stream), configObj) {
				allStreams = append(allStreams, aws.String(stream))
			}
		}
	}

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return allStreams, nil
}

func shouldIncludeKinesisStream(streamName *string, configObj config.Config) bool {
	if streamName == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(streamName),
		configObj.KinesisStream.IncludeRule.NamesRegExp,
		configObj.KinesisStream.ExcludeRule.NamesRegExp,
	)
}

func nukeAllKinesisStreams(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return err
	}
	svc := kinesis.NewFromConfig(cfg)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No Kinesis Streams to nuke in region: %s", region)
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on KinesisStream.MaxBatchSize, however we add a guard here to warn users when the batching fails and
	// has a chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the
	// limit here because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many Kinesis Streams at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyStreamsErr{}
	}

	// There is no bulk delete Kinesis Stream API, so we delete the batch of Kinesis Streams concurrently
	// using go routines.
	logging.Logger.Debugf("Deleting Kinesis Streams in region: %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, streamName := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteKinesisStreamAsync(wg, errChans[i], svc, streamName, region)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	// NOTE: We ignore OperationAbortedException which is thrown when there is an eventual consistency issue, where
	// cloud-nuke picks up a Stream that is already requested to be deleted.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "OperationAbortedException" {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking Kinesis Stream",
				}, map[string]interface{}{
					"region": *session.Config.Region,
				})
				allErrs = multierror.Append(allErrs, err)
			}
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func deleteKinesisStreamAsync(
	wg *sync.WaitGroup,
	errChan chan error,
	svc *kinesis.Client,
	streamName *string,
	region string,
) {
	defer wg.Done()
	input := &kinesis.DeleteStreamInput{StreamName: streamName}
	_, err := svc.DeleteStream(context.TODO(), input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(streamName),
		ResourceType: "Kinesis Stream",
		Error:        err,
	}
	report.Record(e)

	errChan <- err

	streamNameStr := aws.StringValue(streamName)
	if err == nil {
		logging.Logger.Debugf("[OK] Kinesis Stream %s delete in %s", streamNameStr, region)
	} else {
		logging.Logger.Debugf("[Failed] Error deleting Kinesis Stream %s in %s: %s", streamNameStr, region, err)
	}
}

// Custom errors

type TooManyStreamsErr struct{}

func (err TooManyStreamsErr) Error() string {
	return "Too many Streams requested at once."
}

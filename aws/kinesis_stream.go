package aws

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllKinesisStreams(session *session.Session, configObj config.Config) ([]*string, error) {
	svc := kinesis.New(session)

	allStreams := []*string{}
	err := svc.ListStreamsPages(
		&kinesis.ListStreamsInput{},
		func(page *kinesis.ListStreamsOutput, lastPage bool) bool {
			for _, streamName := range page.StreamNames {
				if shouldIncludeKinesisStream(streamName, configObj) {
					allStreams = append(allStreams, streamName)
				}
			}
			return !lastPage
		},
	)
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
	svc := kinesis.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Infof("No Kinesis Streams to nuke in region: %s", region)
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
	logging.Logger.Infof("Deleting Kinesis Streams in region: %s", region)
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
	svc *kinesis.Kinesis,
	streamName *string,
	region string,
) {
	defer wg.Done()
	input := &kinesis.DeleteStreamInput{StreamName: streamName}
	_, err := svc.DeleteStream(input)
	errChan <- err

	streamNameStr := aws.StringValue(streamName)
	if err == nil {
		logging.Logger.Infof("[OK] Kinesis Stream %s delete in %s", streamNameStr, region)
	} else {
		logging.Logger.Errorf("[Failed] Error deleting Kinesis Stream %s in %s: %s", streamNameStr, region, err)
	}
}

// Custom errors

type TooManyStreamsErr struct{}

func (err TooManyStreamsErr) Error() string {
	return "Too many Streams requested at once."
}

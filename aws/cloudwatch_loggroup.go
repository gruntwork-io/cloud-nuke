package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllCloudWatchLogGroups(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := cloudwatchlogs.New(session)

	allLogGroups := []*string{}
	err := svc.DescribeLogGroupsPages(
		&cloudwatchlogs.DescribeLogGroupsInput{},
		func(page *cloudwatchlogs.DescribeLogGroupsOutput, lastPage bool) bool {
			for _, logGroup := range page.LogGroups {
				if shouldIncludeCloudWatchLogGroup(logGroup, excludeAfter, configObj) {
					allLogGroups = append(allLogGroups, logGroup.LogGroupName)
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return allLogGroups, nil
}

func shouldIncludeCloudWatchLogGroup(logGroup *cloudwatchlogs.LogGroup, excludeAfter time.Time, configObj config.Config) bool {
	if logGroup == nil {
		return false
	}

	if logGroup.CreationTime != nil {
		// Convert milliseconds since epoch to time.Time object
		creationTime := time.Unix(0, aws.Int64Value(logGroup.CreationTime)*int64(time.Millisecond))
		if excludeAfter.Before(creationTime) {
			return false
		}
	}

	return config.ShouldInclude(
		aws.StringValue(logGroup.LogGroupName),
		configObj.CloudWatchLogGroup.IncludeRule.NamesRegExp,
		configObj.CloudWatchLogGroup.ExcludeRule.NamesRegExp,
	)
}

func nukeAllCloudWatchLogGroups(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)
	svc := cloudwatchlogs.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No CloudWatch Log Groups to nuke in region %s", *session.Config.Region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on CloudWatchLogGroup.MaxBatchSize, however we add a guard here to warn users when the batching fails and
	// has a chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the
	// limit here because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many CloudWatch LogGroups at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyLogGroupsErr{}
	}

	// There is no bulk delete CloudWatch Log Group API, so we delete the batch of CloudWatch Log Groups concurrently
	// using go routines.
	logging.Logger.Debugf("Deleting CloudWatch Log Groups in region %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, logGroupName := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteCloudWatchLogGroupAsync(wg, errChans[i], svc, logGroupName, region)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	// NOTE: We ignore OperationAbortedException which is thrown when there is an eventual consistency issue, where
	// cloud-nuke picks up a Log Group that is already requested to be deleted.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "OperationAbortedException" {
				allErrs = multierror.Append(allErrs, err)
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking Cloudwatch Log Group",
				}, map[string]interface{}{
					"region": *session.Config.Region,
				})
			}
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

// deleteCloudWatchLogGroupAsync deletes the provided Log Group asynchronously in a goroutine, using wait groups for
// concurrency control and a return channel for errors.
func deleteCloudWatchLogGroupAsync(
	wg *sync.WaitGroup,
	errChan chan error,
	svc *cloudwatchlogs.CloudWatchLogs,
	logGroupName *string,
	region string,
) {
	defer wg.Done()
	input := &cloudwatchlogs.DeleteLogGroupInput{LogGroupName: logGroupName}
	_, err := svc.DeleteLogGroup(input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(logGroupName),
		ResourceType: "CloudWatch Log Group",
		Error:        err,
	}
	report.Record(e)

	errChan <- err

	logGroupNameStr := aws.StringValue(logGroupName)
	if err == nil {
		logging.Logger.Debugf("[OK] CloudWatch Log Group %s deleted in %s", logGroupNameStr, region)
	} else {
		logging.Logger.Debugf("[Failed] Error deleting CloudWatch Log Group %s in %s: %s", logGroupNameStr, region, err)
	}
}

// Custom errors

type TooManyLogGroupsErr struct{}

func (err TooManyLogGroupsErr) Error() string {
	return "Too many LogGroups requested at once."
}

package resources

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	goerr "github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (csr *CloudWatchLogGroups) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var allLogGroups []*string

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(csr.Client, &cloudwatchlogs.DescribeLogGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, goerr.WithStackTrace(err)
		}

		for _, logGroup := range page.LogGroups {
			var creationTime *time.Time
			if logGroup.CreationTime != nil {
				// Convert milliseconds since epoch to time.Time object
				creationTime = aws.Time(time.Unix(0, aws.ToInt64(logGroup.CreationTime)*int64(time.Millisecond)))
			}

			if configObj.CloudWatchLogGroup.ShouldInclude(config.ResourceValue{
				Name: logGroup.LogGroupName,
				Time: creationTime,
			}) {
				allLogGroups = append(allLogGroups, logGroup.LogGroupName)
			}
		}
	}

	return allLogGroups, nil
}

func (csr *CloudWatchLogGroups) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No CloudWatch Log Groups to nuke in region %s", csr.Region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on CloudWatchLogGroup.MaxBatchSize, however we add a guard here to warn users when the batching fails and
	// has a chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the
	// limit here because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Errorf("Nuking too many CloudWatch LogGroups at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyLogGroupsErr{}
	}

	// There is no bulk delete CloudWatch Log Group API, so we delete the batch of CloudWatch Log Groups concurrently
	// using go routines.
	logging.Debugf("Deleting CloudWatch Log Groups in region %s", csr.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, logGroupName := range identifiers {
		errChans[i] = make(chan error, 1)
		go csr.deleteAsync(wg, errChans[i], logGroupName)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	// NOTE: We ignore OperationAbortedException which is thrown when there is an eventual consistency issue, where
	// cloud-nuke picks up a Log Group that is already requested to be deleted.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			var awsErr *types.OperationAbortedException
			if !errors.As(err, &awsErr) {
				allErrs = multierror.Append(allErrs, err)
			}
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return goerr.WithStackTrace(finalErr)
	}
	return nil
}

// deleteAsync deletes the provided Log Group asynchronously in a goroutine, using wait groups for
// concurrency control and a return channel for errors.
func (csr *CloudWatchLogGroups) deleteAsync(wg *sync.WaitGroup, errChan chan error, logGroupName *string) {
	defer wg.Done()
	input := &cloudwatchlogs.DeleteLogGroupInput{LogGroupName: logGroupName}
	_, err := csr.Client.DeleteLogGroup(csr.Context, input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.ToString(logGroupName),
		ResourceType: "CloudWatch Log Group",
		Error:        err,
	}
	report.Record(e)

	errChan <- err

	logGroupNameStr := aws.ToString(logGroupName)
	if err == nil {
		logging.Debugf("[OK] CloudWatch Log Group %s deleted in %s", logGroupNameStr, csr.Region)
	} else {
		logging.Debugf("[Failed] Error deleting CloudWatch Log Group %s in %s: %s", logGroupNameStr, csr.Region, err)
	}
}

// Custom errors

type TooManyLogGroupsErr struct{}

func (err TooManyLogGroupsErr) Error() string {
	return "Too many LogGroups requested at once."
}

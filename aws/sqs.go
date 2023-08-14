package aws

import (
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of SQS Queue URLs
func (sq *SqsQueue) getAll(configObj config.Config) ([]*string, error) {
	result := []*string{}
	paginator := func(output *sqs.ListQueuesOutput, lastPage bool) bool {
		result = append(result, output.QueueUrls...)
		return !lastPage
	}

	param := &sqs.ListQueuesInput{
		MaxResults: awsgo.Int64(10),
	}
	err := sq.Client.ListQueuesPages(param, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var urls []*string

	for _, queue := range result {
		param := &sqs.GetQueueAttributesInput{
			QueueUrl:       queue,
			AttributeNames: awsgo.StringSlice([]string{"CreatedTimestamp"}),
		}
		queueAttributes, err := sq.Client.GetQueueAttributes(param)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Convert string timestamp to int64
		createdAt := *queueAttributes.Attributes["CreatedTimestamp"]
		createdAtTime, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Compare time as int64
		if configObj.SQS.ShouldInclude(config.ResourceValue{
			Name: queue,
			Time: &createdAtTime,
		}) {
			urls = append(urls, queue)
		}
	}

	return urls, nil
}

// Deletes all Elastic Load Balancers
func (sq *SqsQueue) nukeAll(urls []*string) error {
	if len(urls) == 0 {
		logging.Logger.Debugf("No SQS Queues to nuke in region %s", sq.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all SQS Queues in region %s", sq.Region)
	var deletedUrls []*string

	for _, url := range urls {
		params := &sqs.DeleteQueueInput{
			QueueUrl: url,
		}

		_, err := sq.Client.DeleteQueue(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(url),
			ResourceType: "SQS Queue",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking SQS Queue",
			}, map[string]interface{}{
				"region": sq.Region,
			})
		} else {
			deletedUrls = append(deletedUrls, url)
			logging.Logger.Debugf("Deleted SQS Queue: %s", *url)
		}
	}

	logging.Logger.Debugf("[OK] %d SQS Queue(s) deleted in %s", len(deletedUrls), sq.Region)

	return nil
}

package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"strconv"
	"time"
)

// Returns a formatted string of SQS Queue URLs
func (sq *SqsQueue) getAll(c context.Context, configObj config.Config) ([]*string, error) {
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

		// Convert string timestamp to int64 and then to time.Time
		createdAt := *queueAttributes.Attributes["CreatedTimestamp"]
		createdAtInt, err := strconv.ParseInt(createdAt, 10, 64)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		createdAtTime := time.Unix(createdAtInt, 0)

		if configObj.SQS.ShouldInclude(config.ResourceValue{
			Name: queue,
			Time: &createdAtTime,
		}) {
			urls = append(urls, queue)
		}
	}

	return urls, nil
}

// Deletes all Queues
func (sq *SqsQueue) nukeAll(urls []*string) error {
	if len(urls) == 0 {
		logging.Debugf("No SQS Queues to nuke in region %s", sq.Region)
		return nil
	}

	logging.Debugf("Deleting all SQS Queues in region %s", sq.Region)
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
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedUrls = append(deletedUrls, url)
			logging.Debugf("Deleted SQS Queue: %s", *url)
		}
	}

	logging.Debugf("[OK] %d SQS Queue(s) deleted in %s", len(deletedUrls), sq.Region)

	return nil
}

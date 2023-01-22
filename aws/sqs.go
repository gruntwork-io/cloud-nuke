package aws

import (
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of SQS Queue URLs
func getAllSqsQueue(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := sqs.New(session)

	result := []*string{}
	paginator := func(output *sqs.ListQueuesOutput, lastPage bool) bool {
		result = append(result, output.QueueUrls...)
		return !lastPage
	}

	param := &sqs.ListQueuesInput{
		MaxResults: awsgo.Int64(10),
	}
	err := svc.ListQueuesPages(param, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var urls []*string

	for _, queue := range result {
		param := &sqs.GetQueueAttributesInput{
			QueueUrl:       queue,
			AttributeNames: awsgo.StringSlice([]string{"CreatedTimestamp"}),
		}
		queueAttributes, err := svc.GetQueueAttributes(param)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Convert string timestamp to int64
		createdAt := *queueAttributes.Attributes["CreatedTimestamp"]
		createdAtInt, err := strconv.ParseInt(createdAt, 10, 64)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Compare time as int64
		if excludeAfter.Unix() > createdAtInt {
			urls = append(urls, queue)
		}
	}

	return urls, nil
}

// Deletes all Elastic Load Balancers
func nukeAllSqsQueues(session *session.Session, urls []*string) error {
	svc := sqs.New(session)

	if len(urls) == 0 {
		logging.Logger.Debugf("No SQS Queues to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all SQS Queues in region %s", *session.Config.Region)
	var deletedUrls []*string

	for _, url := range urls {
		params := &sqs.DeleteQueueInput{
			QueueUrl: url,
		}

		_, err := svc.DeleteQueue(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(url),
			ResourceType: "SQS Queue",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
		} else {
			deletedUrls = append(deletedUrls, url)
			logging.Logger.Debugf("Deleted SQS Queue: %s", *url)
		}
	}

	logging.Logger.Debugf("[OK] %d SQS Queue(s) deleted in %s", len(deletedUrls), *session.Config.Region)

	return nil
}

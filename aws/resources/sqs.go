package resources

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of SQS Queue URLs
func (sq *SqsQueue) getAll(ctx context.Context, configObj config.Config) ([]*string, error) {
	var result []*string

	paginator := sqs.NewListQueuesPaginator(sq.Client, &sqs.ListQueuesInput{
		MaxResults: aws.Int32(10),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		result = append(result, aws.StringSlice(page.QueueUrls)...)
	}

	var urls []*string

	for _, queue := range result {
		param := &sqs.GetQueueAttributesInput{
			QueueUrl:       queue,
			AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameCreatedTimestamp},
		}
		queueAttributes, err := sq.Client.GetQueueAttributes(ctx, param)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Convert string timestamp to int64 and then to time.Time
		createdAt, ok := queueAttributes.Attributes["CreatedTimestamp"]
		if !ok {
			return nil, errors.WithStackTrace(fmt.Errorf("expected to find CreatedTimestamp attribute"))
		}
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

		_, err := sq.Client.DeleteQueue(sq.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(url),
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

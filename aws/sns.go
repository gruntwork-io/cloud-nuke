package aws

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// getAllSNSTopics returns a list of all SNS topics in the region, filtering the name by the config
// The SQS APIs do not return a creation date, therefore we tag the resources with a first seen time when the topic first appears. We then
// use that tag to measure the excludeAfter time duration, and determine whether to nuke the resource based on that.
func getAllSNSTopics(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	ctx := context.TODO()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}
	svc := sns.NewFromConfig(cfg)

	snsTopics := []*string{}

	paginator := sns.NewListTopicsPaginator(svc, nil)

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return []*string{}, errors.WithStackTrace(err)
		}
		for _, topic := range resp.Topics {
			firstSeenTime, err := getFirstSeenSNSTopicTag(ctx, svc, *topic.TopicArn, firstSeenTagKey)
			if err != nil {
				logging.Logger.Errorf("Unable to retrieve tags for SNS Topic: %s, with error: %s", *topic.TopicArn, err)
				return nil, err
			}

			if firstSeenTime == nil {
				now := time.Now().UTC()
				firstSeenTime = &now
				if err := setFirstSeenSNSTopicTag(ctx, svc, *topic.TopicArn, firstSeenTagKey, now); err != nil {
					logging.Logger.Errorf("Unable to apply first seen tag SNS Topic: %s, with error: %s", *topic.TopicArn, err)
					return nil, err
				}
			}

			if shouldIncludeSNS(*topic.TopicArn, excludeAfter, *firstSeenTime, configObj) {
				snsTopics = append(snsTopics, topic.TopicArn)
			}
		}
	}
	return snsTopics, nil
}

// getFirstSeenSNSTopicTag will retrive the time that the topic was first seen, otherwise returning nil if the topic has not been
// seen before.
func getFirstSeenSNSTopicTag(ctx context.Context, svc *sns.Client, topicArn, key string) (*time.Time, error) {
	response, err := svc.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{
		ResourceArn: &topicArn,
	})
	if err != nil {
		return nil, err
	}

	for i := range response.Tags {
		if *response.Tags[i].Key == key {
			firstSeenTime, err := time.Parse(firstSeenTimeFormat, *response.Tags[i].Value)
			if err != nil {
				return nil, err
			}

			return &firstSeenTime, nil
		}
	}

	return nil, nil
}

// setFirstSeenSNSTopic will append a tag to the SNS Topic that details the first seen time.
func setFirstSeenSNSTopicTag(ctx context.Context, svc *sns.Client, topicArn, key string, value time.Time) error {
	timeValue := value.Format(firstSeenTimeFormat)

	_, err := svc.TagResource(
		ctx,
		&sns.TagResourceInput{
			ResourceArn: &topicArn,
			Tags: []types.Tag{
				{
					Key:   &key,
					Value: &timeValue,
				},
			},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// shouldIncludeSNS checks if the SNS topic should be included in the nuke list based on the config
func shouldIncludeSNS(topicArn string, excludeAfter, firstSeenTime time.Time, configObj config.Config) bool {
	// a topic arn is of the form arn:aws:sns:us-east-1:123456789012:MyTopic
	// so we can search for the index of the last colon, then slice the string to get the topic name
	nameIndex := strings.LastIndex(topicArn, ":")
	topicName := topicArn[nameIndex+1:]

	if excludeAfter.Before(firstSeenTime) {
		return false
	}

	return config.ShouldInclude(topicName, configObj.SNS.IncludeRule.NamesRegExp, configObj.SNS.ExcludeRule.NamesRegExp)
}

func nukeAllSNSTopics(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return errors.WithStackTrace(err)
	}
	svc := sns.NewFromConfig(cfg)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No SNS Topics to nuke in region %s", region)
	}

	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many SNS Topics (100): halting to avoid hitting AWS API rate limiting")
		return TooManySNSTopicsErr{}
	}

	// There is no bulk delete SNS API, so we delete the batch of SNS Topics concurrently using goroutines
	logging.Logger.Debugf("Deleting SNS Topics in region %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, topicArn := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteSNSTopicAsync(wg, errChans[i], svc, topicArn, region)
	}
	wg.Wait()

	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking SNS Topic",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func deleteSNSTopicAsync(wg *sync.WaitGroup, errChan chan error, svc *sns.Client, topicArn *string, region string) {
	defer wg.Done()

	deleteParam := &sns.DeleteTopicInput{
		TopicArn: topicArn,
	}

	logging.Logger.Debugf("Deleting SNS Topic (arn=%s) in region: %s", aws.StringValue(topicArn), region)

	_, err := svc.DeleteTopic(context.TODO(), deleteParam)

	errChan <- err

	// Record status of this resource
	e := report.Entry{
		Identifier:   *topicArn,
		ResourceType: "SNS Topic",
		Error:        err,
	}
	report.Record(e)

	if err == nil {
		logging.Logger.Debugf("[OK] Deleted SNS Topic (arn=%s) in region: %s", aws.StringValue(topicArn), region)
	} else {
		logging.Logger.Debugf("[Failed] Error deleting SNS Topic (arn=%s) in %s", aws.StringValue(topicArn), region)
	}
}

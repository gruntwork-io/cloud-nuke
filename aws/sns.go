package aws

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"strings"
	"sync"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// getAllSNSTopics returns a list of all SNS topics in the region, filtering the name by the config
// The SQS APIs do not return a creation date, therefore we tag the resources with a first seen time when the topic first appears. We then
// use that tag to measure the excludeAfter time duration, and determine whether to nuke the resource based on that.
func (s SNSTopic) getAll(configObj config.Config) ([]*string, error) {

	var snsTopics []*string
	err := s.Client.ListTopicsPages(&sns.ListTopicsInput{}, func(page *sns.ListTopicsOutput, lastPage bool) bool {
		for _, topic := range page.Topics {
			firstSeenTime, err := s.getFirstSeenSNSTopicTag(*topic.TopicArn)
			if err != nil {
				logging.Logger.Errorf(
					"Unable to retrieve tags for SNS Topic: %s, with error: %s", *topic.TopicArn, err)
				continue
			}

			if firstSeenTime == nil {
				now := time.Now().UTC()
				firstSeenTime = &now
				if err := s.setFirstSeenSNSTopicTag(*topic.TopicArn, now); err != nil {
					logging.Logger.Errorf(
						"Unable to apply first seen tag SNS Topic: %s, with error: %s", *topic.TopicArn, err)
					continue
				}
			}

			// a topic arn is of the form arn:aws:sns:us-east-1:123456789012:MyTopic
			// so we can search for the index of the last colon, then slice the string to get the topic name
			nameIndex := strings.LastIndex(*topic.TopicArn, ":")
			topicName := (*topic.TopicArn)[nameIndex+1:]
			if configObj.SNS.ShouldInclude(config.ResourceValue{
				Time: firstSeenTime,
				Name: &topicName,
			}) {
				snsTopics = append(snsTopics, topic.TopicArn)
			}
		}

		return !lastPage
	})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return snsTopics, nil
}

// getFirstSeenSNSTopicTag will retrive the time that the topic was first seen, otherwise returning nil if the topic has not been
// seen before.
func (s SNSTopic) getFirstSeenSNSTopicTag(topicArn string) (*time.Time, error) {
	response, err := s.Client.ListTagsForResource(&sns.ListTagsForResourceInput{
		ResourceArn: &topicArn,
	})
	if err != nil {
		return nil, err
	}

	for i := range response.Tags {
		if *response.Tags[i].Key == firstSeenTagKey {
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
func (s SNSTopic) setFirstSeenSNSTopicTag(topicArn string, value time.Time) error {
	timeValue := value.Format(firstSeenTimeFormat)

	_, err := s.Client.TagResource(
		&sns.TagResourceInput{
			ResourceArn: &topicArn,
			Tags: []*sns.Tag{
				{
					Key:   aws.String(firstSeenTagKey),
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

func (s SNSTopic) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Logger.Debugf("No SNS Topics to nuke in region %s", s.Region)
	}

	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many SNS Topics (100): halting to avoid hitting AWS API rate limiting")
		return TooManySNSTopicsErr{}
	}

	// There is no bulk delete SNS API, so we delete the batch of SNS Topics concurrently using goroutines
	logging.Logger.Debugf("Deleting SNS Topics in region %s", s.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, topicArn := range identifiers {
		errChans[i] = make(chan error, 1)
		go s.deleteAsync(wg, errChans[i], topicArn)
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
				"region": s.Region,
			})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func (s SNSTopic) deleteAsync(wg *sync.WaitGroup, errChan chan error, topicArn *string) {
	defer wg.Done()

	deleteParam := &sns.DeleteTopicInput{
		TopicArn: topicArn,
	}

	logging.Logger.Debugf("Deleting SNS Topic (arn=%s) in region: %s", aws.StringValue(topicArn), s.Region)

	_, err := s.Client.DeleteTopic(deleteParam)

	errChan <- err

	// Record status of this resource
	e := report.Entry{
		Identifier:   *topicArn,
		ResourceType: "SNS Topic",
		Error:        err,
	}
	report.Record(e)

	if err == nil {
		logging.Logger.Debugf("[OK] Deleted SNS Topic (arn=%s) in region: %s", aws.StringValue(topicArn), s.Region)
	} else {
		logging.Logger.Debugf("[Failed] Error deleting SNS Topic (arn=%s) in %s", aws.StringValue(topicArn), s.Region)
	}
}

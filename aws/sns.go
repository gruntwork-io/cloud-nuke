package aws

import (
	"context"
	"strings"
	"sync"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// getAllSNSTopics returns a list of all SNS topics in the region, filtering the name by the config
// It is not possible to filter the SNS Topics by creation time, as there are no SQS APIs that support returning the
// creation time of the topic.
func getAllSNSTopics(session *session.Session, configObj config.Config) ([]*string, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}
	svc := sns.NewFromConfig(cfg)

	snsTopics := []*string{}

	paginator := sns.NewListTopicsPaginator(svc, nil)

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(context.TODO())
		if err != nil {
			return []*string{}, errors.WithStackTrace(err)
		}
		for _, topic := range resp.Topics {
			if shouldIncludeSNS(aws.StringValue(topic.TopicArn), configObj) {
				snsTopics = append(snsTopics, topic.TopicArn)
			}
		}
	}
	return snsTopics, nil
}

// shouldIncludeSNS checks if the SNS topic should be included in the nuke list based on the config
func shouldIncludeSNS(topicArn string, configObj config.Config) bool {
	// a topic arn is of the form arn:aws:sns:us-east-1:123456789012:MyTopic
	// so we can search for the index of the last colon, then slice the string to get the topic name
	nameIndex := strings.LastIndex(topicArn, ":")
	topicName := topicArn[nameIndex+1:]

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

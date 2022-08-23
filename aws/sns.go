package aws

import (
	"context"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllSNSTopics(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}
	svc := sns.NewFromConfig(cfg)

	allSNSTopics := []*string{}

	paginator := sns.NewListTopicsPaginator(svc, nil)

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(context.TODO())
		if err != nil {
			return []*string{}, errors.WithStackTrace(err)
		}
		for _, topic := range resp.Topics {
			allSNSTopics = append(allSNSTopics, topic.TopicArn)
		}
	}
	return allSNSTopics, nil
}

func nukeAllSNSTopics(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(aws.StringValue(session.Config.Region)))
	if err != nil {
		return errors.WithStackTrace(err)
	}
	svc := sns.NewFromConfig(cfg)

	if len(identifiers) == 0 {
		logging.Logger.Infof("No SNS Topics to nuke in region %s", region)
	}

	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many SNS Topics (100): halting to avoid hitting AWS API rate limiting")
		return TooManySNSTopicsErr{}
	}

	// There is no bulk delete SNS API, so we delete the batch of SNS Topics concurrently using goroutines
	logging.Logger.Infof("Deleting SNS Topics in region %s", region)
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
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func deleteSNSTopicAsync(wg *sync.WaitGroup, errChan chan error, svc *sns.Client, topicArn *string, region string) {
	var allErrs *multierror.Error

	defer wg.Done()
	defer func() { errChan <- allErrs.ErrorOrNil() }()

	deleteParam := &sns.DeleteTopicInput{
		TopicArn: topicArn,
	}

	logging.Logger.Infof("Deleting SNS Topic (arn=%s) in region: %s", aws.StringValue(topicArn), region)

	_, err := svc.DeleteTopic(context.TODO(), deleteParam)
	if err != nil {
		allErrs = multierror.Append(allErrs, err)
	} else {
		logging.Logger.Infof("[OK] Deleted SNS Topic (arn=%s) in region: %s", aws.StringValue(topicArn), region)
	}

	if err == nil {
		logging.Logger.Infof("[OK] SNS Topic (arn=%s) deleted in %s", aws.StringValue(topicArn), region)
	} else {
		logging.Logger.Errorf("[Failed] Error deleting SNS Topic (arn=%s) in %s", aws.StringValue(topicArn), region)
	}
}

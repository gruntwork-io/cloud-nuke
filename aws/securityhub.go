package aws

import (
	"strings"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllSecurityHubInsights(session *session.Session, region string) ([]*string, error) {
	svc := securityhub.New(session)
	result, err := svc.GetInsights(&securityhub.GetInsightsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	names = append(names, awsgo.String("disable-securityhub"))
	for _, insight := range result.Insights {
		names = append(names, insight.Name)
	}

	return names, nil
}

func deleteSecurityHubInsight(svc *securityhub.SecurityHub, insightArn *string) error {
	_, err := svc.DeleteInsight(&securityhub.DeleteInsightInput{
		InsightArn: insightArn,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func nukeAllSecurityHubInsights(session *session.Session, insights []*string) error {
	if len(insights) == 0 {
		logging.Logger.Info("No Security Hub Insights to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all Security Hub Insights")

	deletedinsights := 0
	svc := securityhub.New(session)
	multiErr := new(multierror.Error)

	for _, name := range insights {
		if *name == "disable-securityhub" {
			continue
		}
		err := deleteSecurityHubInsight(svc, name)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		} else {
			deletedinsights++
			logging.Logger.Infof("Deleted Security Hub Insight: %s", *name)
		}
	}

	logging.Logger.Infof("[OK] %d Security Hub Insights(s) terminated", deletedinsights)
	logging.Logger.Infof("[OK] Security Hub Disabled")

	return multiErr.ErrorOrNil()
}

func checkSecurityHubEnabled(session *session.Session) (bool, error) {
	svc := securityhub.New(session)
	_, err := svc.GetEnabledStandards(&securityhub.GetEnabledStandardsInput{})
	if err != nil && !strings.Contains(err.Error(), "InvalidAccessException") {
		return false, err
	}

	if err != nil && strings.Contains(err.Error(), "InvalidAccessException") {
		logging.Logger.Debugf("[Skipped] SecurityHub is not enabled for the region")
		return false, nil
	}

	return true, nil
}

func disableSecurityHub(session *session.Session) error {
	svc := securityhub.New(session)
	_, err := svc.DisableSecurityHub(&securityhub.DisableSecurityHubInput{})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return err
	}

	return nil
}

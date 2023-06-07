package aws

import (
	"time"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
)

func getAllCodeDeployApplications(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := codedeploy.New(session)

	codeDeployApplicationsFilteredByName := []*string{}

	err := svc.ListApplicationsPages(&codedeploy.ListApplicationsInput{}, func(page *codedeploy.ListApplicationsOutput, lastPage bool) bool {
		for _, application := range page.Applications {
			// Check if the CodeDeploy Application should be excluded by name as that information is available to us here.
			// CreationDate is not available in the ListApplications API call, so we can't filter by that here, but we do filter by it later.
			// By filtering the name here, we can reduce the number of BatchGetApplication API calls we have to make.
			if config.ShouldInclude(*application, configObj.CodeDeployApplications.IncludeRule.NamesRegExp, configObj.CodeDeployApplications.ExcludeRule.NamesRegExp) {
				codeDeployApplicationsFilteredByName = append(codeDeployApplicationsFilteredByName, application)
			}
		}
		return !lastPage
	})
	if err != nil {
		return nil, err
	}

	// Check if the CodeDeploy Application should be excluded by CreationDate and return.
	// We have to do this after the ListApplicationsPages API call because CreationDate is not available in that call.
	return batchDescribeAndFilterCodeDeployApplications(session, codeDeployApplicationsFilteredByName, excludeAfter)
}

// batchDescribeAndFilterCodeDeployApplications - Describe the CodeDeploy Applications and filter out the ones that should be excluded by CreationDate.
func batchDescribeAndFilterCodeDeployApplications(session *session.Session, identifiers []*string, excludeAfter time.Time) ([]*string, error) {
	svc := codedeploy.New(session)

	applicationNames := []*string{}

	// BatchGetApplications can only take 100 identifiers at a time, so we have to break up the identifiers into chunks of 100.
	for i := 0; i < len(identifiers); i += 100 {

		// Determine the end index of the identifiers slice.
		var end int
		if len(identifiers)-i < 100 {
			end = len(identifiers)
		} else {
			end = i + 100
		}

		resp, err := svc.BatchGetApplications(&codedeploy.BatchGetApplicationsInput{ApplicationNames: identifiers[i:end]})
		if err != nil {
			return nil, err
		}

		// Check if the CodeDeploy Application should be excluded by CreationDate.
		for j := range resp.ApplicationsInfo {
			if resp.ApplicationsInfo[j] != nil && resp.ApplicationsInfo[j].CreateTime != nil && excludeAfter.Before(*resp.ApplicationsInfo[j].CreateTime) {
				applicationNames = append(applicationNames, resp.ApplicationsInfo[j].ApplicationName)
			}
		}
	}

	return applicationNames, nil
}

func nukeAllCodeDeployApplications(session *session.Session, identifiers []*string) error {
	svc := codedeploy.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No CodeDeploy Applications to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all CodeDeploy Applications in region %s", *session.Config.Region)

	for _, identifier := range identifiers {
		_, err := svc.DeleteApplication(&codedeploy.DeleteApplicationInput{ApplicationName: identifier})
		if err != nil {
			logging.Logger.Errorf("Error deleting CodeDeploy Application %s: %s", *identifier, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking CodeDeploy Application",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			logging.Logger.Debugf("[OK] Deleted CodeDeploy Application: %s", *identifier)
		}
	}

	return nil
}

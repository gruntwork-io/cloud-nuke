package aws

import (
	"time"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
)

func getAllCodeDeployApplications(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
	svc := codedeploy.New(session)

	codeDeployApplicationsFilteredByName := []string{}

	err := svc.ListApplicationsPages(&codedeploy.ListApplicationsInput{}, func(page *codedeploy.ListApplicationsOutput, lastPage bool) bool {
		for _, application := range page.Applications {
			// Check if the CodeDeploy Application should be excluded by name as that information is available to us here.
			// CreationDate is not available in the ListApplications API call, so we can't filter by that here, but we do filter by it later.
			// By filtering the name here, we can reduce the number of BatchGetApplication API calls we have to make.
			if config.ShouldInclude(*application, configObj.CodeDeployApplications.IncludeRule.NamesRegExp, configObj.CodeDeployApplications.ExcludeRule.NamesRegExp) {
				codeDeployApplicationsFilteredByName = append(codeDeployApplicationsFilteredByName, *application)
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
func batchDescribeAndFilterCodeDeployApplications(session *session.Session, identifiers []string, excludeAfter time.Time) ([]string, error) {
	svc := codedeploy.New(session)

	// BatchGetApplications can only take 100 identifiers at a time, so we have to break up the identifiers into chunks of 100.
	batchSize := 100
	applicationNames := []string{}

	for {
		// if there are no identifiers left, then break out of the loop
		if len(identifiers) == 0 {
			break
		}

		// if the batch size is larger than the number of identifiers left, then set the batch size to the number of identifiers left
		if len(identifiers) < batchSize {
			batchSize = len(identifiers)
		}

		// get the next batch of identifiers
		batch := aws.StringSlice(identifiers[:batchSize])
		// then using that batch of identifiers, get the applicationsinfo
		resp, err := svc.BatchGetApplications(
			&codedeploy.BatchGetApplicationsInput{ApplicationNames: batch},
		)
		if err != nil {
			return nil, err
		}

		// for each applicationsinfo, check if it should be excluded by creation date
		for j := range resp.ApplicationsInfo {
			if shouldNukeByCreationTime(resp.ApplicationsInfo[j], excludeAfter) {
				applicationNames = append(applicationNames, *resp.ApplicationsInfo[j].ApplicationName)
			}
		}

		// reduce the identifiers by the batch size we just processed, note that the slice header is mutated here
		identifiers = identifiers[batchSize:]
	}

	return applicationNames, nil
}

// shouldNukeByCreationTime - Check if the CodeDeploy Application should be excluded by CreationDate.
func shouldNukeByCreationTime(applicationInfo *codedeploy.ApplicationInfo, excludeAfter time.Time) bool {
	// If the CreationDate is nil, then we can't filter by it, so we return false as a precaution.
	if applicationInfo == nil || applicationInfo.CreateTime == nil {
		return false
	}

	// If the excludeAfter date is before the CreationDate, then we should not nuke the resource.
	if excludeAfter.Before(*applicationInfo.CreateTime) {
		return false
	}

	// Otherwise, the resource can be safely nuked.
	return true
}

func nukeAllCodeDeployApplications(session *session.Session, identifiers []string) error {
	svc := codedeploy.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No CodeDeploy Applications to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting CodeDeploy Applications in region %s", *session.Config.Region)

	for _, identifier := range identifiers {
		_, err := svc.DeleteApplication(&codedeploy.DeleteApplicationInput{ApplicationName: &identifier})
		if err != nil {
			logging.Logger.Errorf("[Failed] Error deleting CodeDeploy Application %s: %s", identifier, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking CodeDeploy Application",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			logging.Logger.Debugf("[OK] Deleted CodeDeploy Application: %s", identifier)
		}

		// record the status of the nuke attempt
		e := report.Entry{
			Identifier:   identifier,
			ResourceType: "CodeDeploy Application",
			Error:        err,
		}
		report.Record(e)
	}

	return nil
}
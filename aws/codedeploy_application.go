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

	allCodeDeployApplications := []*string{}

	err := svc.ListApplicationsPages(&codedeploy.ListApplicationsInput{}, func(page *codedeploy.ListApplicationsOutput, lastPage bool) bool {
		for _, application := range page.Applications {
			shouldInclude, err := shouldIncludeCodeDeployApplication(svc, application, excludeAfter, configObj)
			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
			}

			if shouldInclude {
				allCodeDeployApplications = append(allCodeDeployApplications, application)
			}
		}
		return !lastPage
	})

	return allCodeDeployApplications, err
}

func shouldIncludeCodeDeployApplication(svc *codedeploy.CodeDeploy, codedeployApplicationName *string, excludeAfter time.Time, configObj config.Config) (bool, error) {
	if codedeployApplicationName == nil {
		return false, nil
	}

	codedeployApplicationDetails, err := describeCodeDeployApplication(svc, codedeployApplicationName)
	if err != nil {
		return false, err
	}

	if codedeployApplicationDetails != nil && codedeployApplicationDetails.CreateTime != nil && excludeAfter.Before(*codedeployApplicationDetails.CreateTime) {
		return false, nil
	}

	return config.ShouldInclude(
		*codedeployApplicationName,
		configObj.CodeDeployApplications.IncludeRule.NamesRegExp,
		configObj.CodeDeployApplications.ExcludeRule.NamesRegExp,
	), nil
}

func describeCodeDeployApplication(svc *codedeploy.CodeDeploy, identifier *string) (*codedeploy.ApplicationInfo, error) {
	resp, err := svc.GetApplication(&codedeploy.GetApplicationInput{ApplicationName: identifier})
	if err != nil {
		return nil, err
	}

	return resp.Application, nil
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

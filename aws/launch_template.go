package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Launch Template Names
func getAllLaunchTemplates(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := ec2.New(session)
	result, err := svc.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var templateNames []*string
	for _, template := range result.LaunchTemplates {
		if shouldIncludeLaunchTemplate(template, excludeAfter, configObj) {
			templateNames = append(templateNames, template.LaunchTemplateName)
		}
	}

	return templateNames, nil
}

func shouldIncludeLaunchTemplate(lt *ec2.LaunchTemplate, excludeAfter time.Time, configObj config.Config) bool {
	if lt == nil {
		return false
	}

	if lt.CreateTime != nil && excludeAfter.Before(*lt.CreateTime) {
		return false
	}

	return config.ShouldInclude(
		awsgo.StringValue(lt.LaunchTemplateName),
		configObj.LaunchTemplate.IncludeRule.NamesRegExp,
		configObj.LaunchTemplate.ExcludeRule.NamesRegExp,
	)
}

// Deletes all Launch Templates
func nukeAllLaunchTemplates(session *session.Session, templateNames []*string) error {
	svc := ec2.New(session)

	if len(templateNames) == 0 {
		logging.Logger.Debugf("No Launch Templates to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Launch Templates in region %s", *session.Config.Region)
	var deletedTemplateNames []*string

	for _, templateName := range templateNames {
		params := &ec2.DeleteLaunchTemplateInput{
			LaunchTemplateName: templateName,
		}

		_, err := svc.DeleteLaunchTemplate(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(templateName),
			ResourceType: "Launch template",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Launch Template",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedTemplateNames = append(deletedTemplateNames, templateName)
			logging.Logger.Debugf("Deleted Launch template: %s", *templateName)
		}
	}

	logging.Logger.Debugf("[OK] %d Launch Template(s) deleted in %s", len(deletedTemplateNames), *session.Config.Region)
	return nil
}

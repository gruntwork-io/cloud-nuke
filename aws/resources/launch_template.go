package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

// Returns a formatted string of Launch Template Names
func (lt *LaunchTemplates) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := lt.Client.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var templateNames []*string
	for _, template := range result.LaunchTemplates {
		if configObj.LaunchTemplate.ShouldInclude(config.ResourceValue{
			Name: template.LaunchTemplateName,
			Time: template.CreateTime,
		}) {
			templateNames = append(templateNames, template.LaunchTemplateName)
		}
	}

	return templateNames, nil
}

// Deletes all Launch Templates
func (lt *LaunchTemplates) nukeAll(templateNames []*string) error {
	if len(templateNames) == 0 {
		logging.Logger.Debugf("No Launch Templates to nuke in region %s", lt.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Launch Templates in region %s", lt.Region)
	var deletedTemplateNames []*string

	for _, templateName := range templateNames {
		params := &ec2.DeleteLaunchTemplateInput{
			LaunchTemplateName: templateName,
		}

		_, err := lt.Client.DeleteLaunchTemplate(params)

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
				"region": lt.Region,
			})
		} else {
			deletedTemplateNames = append(deletedTemplateNames, templateName)
			logging.Logger.Debugf("Deleted Launch template: %s", *templateName)
		}
	}

	logging.Logger.Debugf("[OK] %d Launch Template(s) deleted in %s", len(deletedTemplateNames), lt.Region)
	return nil
}

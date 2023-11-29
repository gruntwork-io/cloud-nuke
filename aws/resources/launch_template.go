package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
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
		logging.Debugf("No Launch Templates to nuke in region %s", lt.Region)
		return nil
	}

	logging.Debugf("Deleting all Launch Templates in region %s", lt.Region)
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
			logging.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Launch Template",
			}, map[string]interface{}{
				"region": lt.Region,
			})
		} else {
			deletedTemplateNames = append(deletedTemplateNames, templateName)
			logging.Debugf("Deleted Launch template: %s", *templateName)
		}
	}

	logging.Debugf("[OK] %d Launch Template(s) deleted in %s", len(deletedTemplateNames), lt.Region)
	return nil
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Launch Template Names
func (lt *LaunchTemplates) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := lt.Client.DescribeLaunchTemplates(lt.Context, &ec2.DescribeLaunchTemplatesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var templateNames []*string
	for _, template := range result.LaunchTemplates {
		// Get tags from the latest version of the launch template
		// $Latest represents the most current configuration and tags
		tags := make(map[string]string)
		versionsResult, err := lt.Client.DescribeLaunchTemplateVersions(c, &ec2.DescribeLaunchTemplateVersionsInput{
			LaunchTemplateId: template.LaunchTemplateId,
			Versions:         []string{"$Latest"},
		})
		if err == nil && len(versionsResult.LaunchTemplateVersions) > 0 {
			for _, tag := range versionsResult.LaunchTemplateVersions[0].LaunchTemplateData.TagSpecifications {
				for _, t := range tag.Tags {
					if t.Key != nil && t.Value != nil {
						tags[*t.Key] = *t.Value
					}
				}
			}
		}

		logging.Debugf("Tags for Launch Template %s: %v", *template.LaunchTemplateName, tags)

		if configObj.LaunchTemplate.ShouldInclude(config.ResourceValue{
			Name: template.LaunchTemplateName,
			Time: template.CreateTime,
			Tags: tags,
		}) {
			templateNames = append(templateNames, template.LaunchTemplateName)
		}
	}

	// checking the nukable permissions
	lt.VerifyNukablePermissions(templateNames, func(id *string) error {
		_, err := lt.Client.DeleteLaunchTemplate(lt.Context, &ec2.DeleteLaunchTemplateInput{
			LaunchTemplateName: id,
			DryRun:             aws.Bool(true),
		})
		return err
	})

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

		if nukable, reason := lt.IsNukable(aws.ToString(templateName)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(templateName), reason)
			continue
		}

		_, err := lt.Client.DeleteLaunchTemplate(lt.Context, &ec2.DeleteLaunchTemplateInput{
			LaunchTemplateName: templateName,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(templateName),
			ResourceType: "Launch template",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Errorf("[Failed] %s", err)
		} else {
			deletedTemplateNames = append(deletedTemplateNames, templateName)
			logging.Debugf("Deleted Launch template: %s", *templateName)
		}
	}

	logging.Debugf("[OK] %d Launch Template(s) deleted in %s", len(deletedTemplateNames), lt.Region)
	return nil
}

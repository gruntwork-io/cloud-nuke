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
		tags := lt.extractTagsFromLatestVersion(c, template.LaunchTemplateId)
		
		logging.Debugf("Tags for Launch Template %s: %v", *template.LaunchTemplateName, tags)

		resourceValue := config.ResourceValue{
			Name: template.LaunchTemplateName,
			Time: template.CreateTime,
			Tags: tags,
		}
		
		if configObj.LaunchTemplate.ShouldInclude(resourceValue) {
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

// extractTagsFromLatestVersion retrieves tags from the latest version of a launch template
func (lt *LaunchTemplates) extractTagsFromLatestVersion(ctx context.Context, templateID *string) map[string]string {
	tags := make(map[string]string)
	
	versionsInput := &ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: templateID,
		Versions:         []string{"$Latest"},
	}
	
	versionsResult, err := lt.Client.DescribeLaunchTemplateVersions(ctx, versionsInput)
	if err != nil || len(versionsResult.LaunchTemplateVersions) == 0 {
		return tags
	}
	
	latestVersion := versionsResult.LaunchTemplateVersions[0]
	if latestVersion.LaunchTemplateData == nil {
		return tags
	}
	
	for _, tagSpec := range latestVersion.LaunchTemplateData.TagSpecifications {
		for _, tag := range tagSpec.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}
	
	return tags
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

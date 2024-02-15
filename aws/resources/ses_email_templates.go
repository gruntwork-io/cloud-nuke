package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of email template names
func (sem *SesEmailTemplates) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	param := &ses.ListTemplatesInput{}

	result, err := sem.Client.ListTemplates(param)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var templates []*string
	for _, template := range result.TemplatesMetadata {
		createdAt := template.CreatedTimestamp
		if configObj.SESEmailTemplates.ShouldInclude(config.ResourceValue{
			Name: template.Name,
			Time: createdAt,
		}) {
			templates = append(templates, template.Name)
		}
	}

	return templates, nil
}

// Deletes all templates
func (sem *SesEmailTemplates) nukeAll(templates []*string) error {
	if len(templates) == 0 {
		logging.Debugf("No SES email templates to nuke in region %s", sem.Region)
		return nil
	}

	logging.Debugf("Deleting all SES email templates in region %s", sem.Region)
	var deletedIds []*string

	for _, template := range templates {
		params := &ses.DeleteTemplateInput{
			TemplateName: template,
		}

		_, err := sem.Client.DeleteTemplate(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(template),
			ResourceType: "SES email templates",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, template)
			logging.Debugf("Deleted SES email templates: %s", *template)
		}
	}

	logging.Debugf("[OK] %d SES email template(s) deleted in %s", len(deletedIds), sem.Region)

	return nil
}

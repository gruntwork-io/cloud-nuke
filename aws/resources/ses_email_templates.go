package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// NewSesEmailTemplates creates a new SES Email Templates resource using the generic resource pattern.
func NewSesEmailTemplates() AwsResource {
	return NewAwsResource(&resource.Resource[*ses.Client]{
		ResourceTypeName: "ses-email-template",
		BatchSize:        maxBatchSize,
		InitClient: func(r *resource.Resource[*ses.Client], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for SES client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ses.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SESEmailTemplates
		},
		Lister: listSesEmailTemplates,
		Nuker:  resource.SimpleBatchDeleter(deleteSesEmailTemplate),
	})
}

// listSesEmailTemplates retrieves all SES email templates that match the config filters.
func listSesEmailTemplates(ctx context.Context, client *ses.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.ListTemplates(ctx, &ses.ListTemplatesInput{})
	if err != nil {
		return nil, err
	}

	var templates []*string
	for _, template := range result.TemplatesMetadata {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: template.Name,
			Time: template.CreatedTimestamp,
		}) {
			templates = append(templates, template.Name)
		}
	}

	return templates, nil
}

// deleteSesEmailTemplate deletes a single SES email template.
func deleteSesEmailTemplate(ctx context.Context, client *ses.Client, templateName *string) error {
	_, err := client.DeleteTemplate(ctx, &ses.DeleteTemplateInput{
		TemplateName: templateName,
	})
	if err != nil {
		return err
	}

	logging.Debugf("Deleted SES email template: %s", aws.ToString(templateName))
	return nil
}

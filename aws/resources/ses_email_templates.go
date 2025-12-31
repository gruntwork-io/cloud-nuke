package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// SesEmailTemplatesAPI defines the interface for SES Email Templates operations.
type SesEmailTemplatesAPI interface {
	ListTemplates(ctx context.Context, params *ses.ListTemplatesInput, optFns ...func(*ses.Options)) (*ses.ListTemplatesOutput, error)
	DeleteTemplate(ctx context.Context, params *ses.DeleteTemplateInput, optFns ...func(*ses.Options)) (*ses.DeleteTemplateOutput, error)
}

// NewSesEmailTemplates creates a new SES Email Templates resource using the generic resource pattern.
func NewSesEmailTemplates() AwsResource {
	return NewAwsResource(&resource.Resource[SesEmailTemplatesAPI]{
		ResourceTypeName: "ses-email-template",
		BatchSize:        maxBatchSize,
		InitClient: func(r *resource.Resource[SesEmailTemplatesAPI], cfg any) {
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
func listSesEmailTemplates(ctx context.Context, client SesEmailTemplatesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var templates []*string
	var nextToken *string

	for {
		result, err := client.ListTemplates(ctx, &ses.ListTemplatesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, template := range result.TemplatesMetadata {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: template.Name,
				Time: template.CreatedTimestamp,
			}) {
				templates = append(templates, template.Name)
			}
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return templates, nil
}

// deleteSesEmailTemplate deletes a single SES email template.
func deleteSesEmailTemplate(ctx context.Context, client SesEmailTemplatesAPI, templateName *string) error {
	_, err := client.DeleteTemplate(ctx, &ses.DeleteTemplateInput{
		TemplateName: templateName,
	})
	return err
}

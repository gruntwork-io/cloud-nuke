package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SESAPI interface {
	ListTemplates(ctx context.Context, params *ses.ListTemplatesInput, optFns ...func(*ses.Options)) (*ses.ListTemplatesOutput, error)
	DeleteTemplate(ctx context.Context, params *ses.DeleteTemplateInput, optFns ...func(*ses.Options)) (*ses.DeleteTemplateOutput, error)
}

// SesEmailTemplates - represents all ses email templates
type SesEmailTemplates struct {
	BaseAwsResource
	Client SESAPI
	Region string
	Ids    []string
}

func (s *SesEmailTemplates) InitV2(cfg aws.Config) {
	s.Client = ses.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (set *SesEmailTemplates) ResourceName() string {
	return "ses-email-template"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (set *SesEmailTemplates) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The names of the ses email templates
func (set *SesEmailTemplates) ResourceIdentifiers() []string {
	return set.Ids
}

func (set *SesEmailTemplates) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SESEmailTemplates
}

func (set *SesEmailTemplates) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := set.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	set.Ids = aws.ToStringSlice(identifiers)
	return set.Ids, nil
}

// Nuke - nuke 'em all!!!
func (set *SesEmailTemplates) Nuke(identifiers []string) error {
	if err := set.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

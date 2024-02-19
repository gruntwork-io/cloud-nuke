package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// SesEmailTemplates - represents all ses email templates
type SesEmailTemplates struct {
	BaseAwsResource
	Client sesiface.SESAPI
	Region string
	Ids    []string
}

func (set *SesEmailTemplates) Init(session *session.Session) {
	set.Client = ses.New(session)
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

func (set *SesEmailTemplates) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := set.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	set.Ids = awsgo.StringValueSlice(identifiers)
	return set.Ids, nil
}

// Nuke - nuke 'em all!!!
func (set *SesEmailTemplates) Nuke(identifiers []string) error {
	if err := set.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

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

// SesConfigurationSet - represents all SES configuartion set
type SesConfigurationSet struct {
	BaseAwsResource
	Client sesiface.SESAPI
	Region string
	Ids    []string
}

func (scs *SesConfigurationSet) Init(session *session.Session) {
	scs.Client = ses.New(session)
}

// ResourceName - the simple name of the aws resource
func (scs *SesConfigurationSet) ResourceName() string {
	return "ses-configuration-set"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (scs *SesConfigurationSet) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the configuration set
func (scs *SesConfigurationSet) ResourceIdentifiers() []string {
	return scs.Ids
}

func (scs *SesConfigurationSet) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SESConfigurationSet
}

func (scs *SesConfigurationSet) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := scs.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	scs.Ids = awsgo.StringValueSlice(identifiers)
	return scs.Ids, nil
}

// Nuke - nuke 'em all!!!
func (scs *SesConfigurationSet) Nuke(identifiers []string) error {
	if err := scs.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

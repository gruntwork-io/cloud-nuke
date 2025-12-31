package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SESConfigurationSet interface {
	ListConfigurationSets(ctx context.Context, params *ses.ListConfigurationSetsInput, optFns ...func(*ses.Options)) (*ses.ListConfigurationSetsOutput, error)
	DeleteConfigurationSet(ctx context.Context, params *ses.DeleteConfigurationSetInput, optFns ...func(*ses.Options)) (*ses.DeleteConfigurationSetOutput, error)
}

// SesConfigurationSet - represents all SES configuartion set
type SesConfigurationSet struct {
	BaseAwsResource
	Client SESConfigurationSet
	Region string
	Ids    []string
}

func (scs *SesConfigurationSet) Init(cfg aws.Config) {
	scs.Client = ses.NewFromConfig(cfg)
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

	scs.Ids = aws.ToStringSlice(identifiers)
	return scs.Ids, nil
}

// Nuke - nuke 'em all!!!
func (scs *SesConfigurationSet) Nuke(ctx context.Context, identifiers []string) error {
	if err := scs.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

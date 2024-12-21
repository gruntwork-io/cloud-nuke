package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SESIdentityAPI interface {
	ListIdentities(ctx context.Context, params *ses.ListIdentitiesInput, optFns ...func(*ses.Options)) (*ses.ListIdentitiesOutput, error)
	DeleteIdentity(ctx context.Context, params *ses.DeleteIdentityInput, optFns ...func(*ses.Options)) (*ses.DeleteIdentityOutput, error)
}

// SesIdentities - represents all SES identities
type SesIdentities struct {
	BaseAwsResource
	Client SESIdentityAPI
	Region string
	Ids    []string
}

func (Sid *SesIdentities) InitV2(cfg aws.Config) {
	Sid.Client = ses.NewFromConfig(cfg)
}

func (Sid *SesIdentities) ResourceName() string {
	return "ses-identity"
}

func (Sid *SesIdentities) MaxBatchSize() int {
	return maxBatchSize
}

func (Sid *SesIdentities) ResourceIdentifiers() []string {
	return Sid.Ids
}

func (Sid *SesIdentities) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SESIdentity
}

func (Sid *SesIdentities) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := Sid.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	Sid.Ids = aws.ToStringSlice(identifiers)
	return Sid.Ids, nil
}

func (Sid *SesIdentities) Nuke(identifiers []string) error {
	if err := Sid.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

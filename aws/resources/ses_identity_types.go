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

// SesIdentities - represents all SES identities
type SesIdentities struct {
	BaseAwsResource
	Client sesiface.SESAPI
	Region string
	Ids    []string
}

func (Sid *SesIdentities) Init(session *session.Session) {
	Sid.Client = ses.New(session)
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

func (Sid *SesIdentities) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := Sid.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	Sid.Ids = awsgo.StringValueSlice(identifiers)
	return Sid.Ids, nil
}

func (Sid *SesIdentities) Nuke(identifiers []string) error {
	if err := Sid.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

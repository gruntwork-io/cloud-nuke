package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/aws/aws-sdk-go/service/macie2/macie2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type MacieMember struct {
	BaseAwsResource
	Client     macie2iface.Macie2API
	Region     string
	AccountIds []string
}

func (mm *MacieMember) Init(session *session.Session) {
	mm.Client = macie2.New(session)
}

func (mm *MacieMember) ResourceName() string {
	return "macie-member"
}

func (mm *MacieMember) ResourceIdentifiers() []string {
	return mm.AccountIds
}

func (mm *MacieMember) MaxBatchSize() int {
	return 10
}

func (mm *MacieMember) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := mm.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	mm.AccountIds = awsgo.StringValueSlice(identifiers)
	return mm.AccountIds, nil
}

func (mm *MacieMember) Nuke(identifiers []string) error {
	if err := mm.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

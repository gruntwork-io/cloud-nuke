package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/macie2/macie2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

type MacieMember struct {
	Client     macie2iface.Macie2API
	Region     string
	AccountIds []string
}

func (mm MacieMember) ResourceName() string {
	return "macie-member"
}

func (mm MacieMember) ResourceIdentifiers() []string {
	return mm.AccountIds
}

func (mm MacieMember) MaxBatchSize() int {
	return 10
}

func (mm MacieMember) Nuke(session *session.Session, identifiers []string) error {
	if err := mm.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

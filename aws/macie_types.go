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

func (r MacieMember) ResourceName() string {
	return "macie-member"
}

func (r MacieMember) ResourceIdentifiers() []string {
	return r.AccountIds
}

func (r MacieMember) MaxBatchSize() int {
	return 10
}

func (r MacieMember) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeMacie(session, identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

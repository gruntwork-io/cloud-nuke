package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type MacieMember struct {
	AccountIds []string
}

func (r MacieMember) ResourceName() string {
	return "macie"
}

func (r MacieMember) ResourceIdentifiers() []string {
	return r.AccountIds
}

func (r MacieMember) MaxBatchSize() int {
	return 10
}

func (r MacieMember) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllMacieMemberAccounts(session, identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

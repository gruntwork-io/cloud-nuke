package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type Macie struct {
	AccountIds []string
}

func (r Macie) ResourceName() string {
	return "macie"
}

func (r Macie) ResourceIdentifiers() []string {
	return r.AccountIds
}

func (r Macie) MaxBatchSize() int {
	return 10
}

func (r Macie) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllMacieAccounts(session, identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

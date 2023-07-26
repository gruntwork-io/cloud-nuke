package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/securityhub/securityhubiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type SecurityHub struct {
	Client  securityhubiface.SecurityHubAPI
	Region  string
	HubArns []string
}

func (sh SecurityHub) ResourceName() string {
	return "security-hub"
}

func (sh SecurityHub) ResourceIdentifiers() []string {
	return sh.HubArns
}

func (sh SecurityHub) MaxBatchSize() int {
	return 5
}

func (sh SecurityHub) Nuke(session *session.Session, identifiers []string) error {
	if err := sh.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

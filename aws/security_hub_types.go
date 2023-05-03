package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type SecurityHub struct {
	HubArns []string
}

func (h SecurityHub) ResourceName() string {
	return "security-hub"
}

func (h SecurityHub) ResourceIdentifiers() []string {
	return h.HubArns
}

func (h SecurityHub) MaxBatchSize() int {
	return 5
}

func (h SecurityHub) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeSecurityHub(session, identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

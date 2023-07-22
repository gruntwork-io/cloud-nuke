package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice/configserviceiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type ConfigServiceRule struct {
	Client    configserviceiface.ConfigServiceAPI
	Region    string
	RuleNames []string
}

func (csr ConfigServiceRule) ResourceName() string {
	return "config-rules"
}

func (csr ConfigServiceRule) ResourceIdentifiers() []string {
	return csr.RuleNames
}

func (csr ConfigServiceRule) MaxBatchSize() int {
	return 200
}

func (csr ConfigServiceRule) Nuke(session *session.Session, identifiers []string) error {
	if err := csr.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

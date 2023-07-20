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

func (c ConfigServiceRule) ResourceName() string {
	return "config-rule"
}

func (c ConfigServiceRule) ResourceIdentifiers() []string {
	return c.RuleNames
}

func (c ConfigServiceRule) MaxBatchSize() int {
	return 200
}

func (c ConfigServiceRule) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllConfigServiceRules(session, identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

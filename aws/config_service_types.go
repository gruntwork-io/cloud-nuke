package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/configservice/configserviceiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ConfigServiceRule struct {
	Client    configserviceiface.ConfigServiceAPI
	Region    string
	RuleNames []string
}

func (csr *ConfigServiceRule) Init(session *session.Session) {
	csr.Client = configservice.New(session)
}

func (csr *ConfigServiceRule) ResourceName() string {
	return "config-rules"
}

func (csr *ConfigServiceRule) ResourceIdentifiers() []string {
	return csr.RuleNames
}

func (csr *ConfigServiceRule) MaxBatchSize() int {
	return 200
}

func (csr *ConfigServiceRule) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := csr.getAll(configObj)
	if err != nil {
		return nil, err
	}

	csr.RuleNames = awsgo.StringValueSlice(identifiers)
	return csr.RuleNames, nil
}

func (csr *ConfigServiceRule) Nuke(identifiers []string) error {
	if err := csr.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

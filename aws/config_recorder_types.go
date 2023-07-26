package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice/configserviceiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type ConfigServiceRecorders struct {
	Client        configserviceiface.ConfigServiceAPI
	Region        string
	RecorderNames []string
}

func (csr ConfigServiceRecorders) ResourceName() string {
	return "config-recorders"
}

func (csr ConfigServiceRecorders) ResourceIdentifiers() []string {
	return csr.RecorderNames
}

func (csr ConfigServiceRecorders) MaxBatchSize() int {
	return 50
}

func (csr ConfigServiceRecorders) Nuke(session *session.Session, configServiceRecorderNames []string) error {
	if err := csr.nukeAll(configServiceRecorderNames); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

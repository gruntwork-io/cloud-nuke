package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type ConfigServiceRecorders struct {
	RecorderNames []string
}

func (u ConfigServiceRecorders) ResourceName() string {
	return "config-recorders"
}

func (u ConfigServiceRecorders) ResourceIdentifiers() []string {
	return u.RecorderNames
}

func (u ConfigServiceRecorders) MaxBatchSize() int {
	return 50
}

func (u ConfigServiceRecorders) Nuke(session *session.Session, configServiceRecorderNames []string) error {
	if err := nukeAllConfigRecorders(session, configServiceRecorderNames); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

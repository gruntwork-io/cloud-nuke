package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice/configserviceiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type ConfigServiceRecorder struct {
	Client        configserviceiface.ConfigServiceAPI
	Region        string
	RecorderNames []string
}

func (u ConfigServiceRecorder) ResourceName() string {
	return "config-recorder"
}

func (u ConfigServiceRecorder) ResourceIdentifiers() []string {
	return u.RecorderNames
}

func (u ConfigServiceRecorder) MaxBatchSize() int {
	return 50
}

func (u ConfigServiceRecorder) Nuke(session *session.Session, configServiceRecorderNames []string) error {
	if err := nukeAllConfigRecorders(session, configServiceRecorderNames); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

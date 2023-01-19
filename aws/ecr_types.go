package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type ECR struct {
	Arns []string
}

func (registry ECR) ResourceName() string {
	return "ecr"
}

func (registry ECR) ResourceIdentifiers() []string {
	return registry.Arns
}

func (registry ECR) MaxBatchSize() int {
	return 50
}

func (registry ECR) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllECRRepositories(session, identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

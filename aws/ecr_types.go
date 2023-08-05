package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	"github.com/gruntwork-io/go-commons/errors"
)

type ECR struct {
	Client          ecriface.ECRAPI
	Region          string
	RepositoryNames []string
}

func (registry ECR) ResourceName() string {
	return "ecr"
}

func (registry ECR) ResourceIdentifiers() []string {
	return registry.RepositoryNames
}

func (registry ECR) MaxBatchSize() int {
	return 50
}

func (registry ECR) Nuke(session *session.Session, identifiers []string) error {
	if err := registry.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

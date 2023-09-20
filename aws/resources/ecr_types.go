package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ECR struct {
	Client          ecriface.ECRAPI
	Region          string
	RepositoryNames []string
}

func (registry *ECR) Init(session *session.Session) {
	registry.Client = ecr.New(session)
}

func (registry *ECR) ResourceName() string {
	return "ecr"
}

func (registry *ECR) ResourceIdentifiers() []string {
	return registry.RepositoryNames
}

func (registry *ECR) MaxBatchSize() int {
	return 50
}

func (registry *ECR) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := registry.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	registry.RepositoryNames = awsgo.StringValueSlice(identifiers)
	return registry.RepositoryNames, nil
}

func (registry *ECR) Nuke(identifiers []string) error {
	if err := registry.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

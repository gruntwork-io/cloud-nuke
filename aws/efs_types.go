package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/efs/efsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type ElasticFileSystem struct {
	Client efsiface.EFSAPI
	Region string
	Ids    []string
}

func (efs ElasticFileSystem) ResourceName() string {
	return "efs"
}

func (efs ElasticFileSystem) ResourceIdentifiers() []string {
	return efs.Ids
}

func (efs ElasticFileSystem) MaxBatchSize() int {
	return 10
}

func (efs ElasticFileSystem) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElasticFileSystems(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

// custom errors

type TooManyElasticFileSystemsErr struct{}

func (err TooManyElasticFileSystemsErr) Error() string {
	return "Too many Elastic FileSystems requested at once."
}

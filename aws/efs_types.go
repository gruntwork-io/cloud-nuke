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

func (ef ElasticFileSystem) ResourceName() string {
	return "efs"
}

func (ef ElasticFileSystem) ResourceIdentifiers() []string {
	return ef.Ids
}

func (ef ElasticFileSystem) MaxBatchSize() int {
	return 10
}

func (ef ElasticFileSystem) Nuke(session *session.Session, identifiers []string) error {
	if err := ef.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// custom errors

type TooManyElasticFileSystemsErr struct{}

func (err TooManyElasticFileSystemsErr) Error() string {
	return "Too many Elastic FileSystems requested at once."
}

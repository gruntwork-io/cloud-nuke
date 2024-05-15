package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/aws/aws-sdk-go/service/s3control/s3controliface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type S3AccessPoint struct {
	BaseAwsResource
	Client       s3controliface.S3ControlAPI
	Region       string
	AccessPoints []string
	AccountID    *string
}

func (ap *S3AccessPoint) Init(session *session.Session) {
	ap.Client = s3control.New(session)
}

func (ap *S3AccessPoint) ResourceName() string {
	return "s3-ap"
}

func (ap *S3AccessPoint) ResourceIdentifiers() []string {
	return ap.AccessPoints
}

func (ap *S3AccessPoint) MaxBatchSize() int {
	return 5
}

func (ap *S3AccessPoint) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.S3AccessPoint
}

func (ap *S3AccessPoint) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ap.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ap.AccessPoints = awsgo.StringValueSlice(identifiers)
	return ap.AccessPoints, nil
}

func (ap *S3AccessPoint) Nuke(identifiers []string) error {
	if err := ap.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

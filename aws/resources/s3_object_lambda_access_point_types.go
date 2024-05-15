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

type S3ObjectLambdaAccessPoint struct {
	BaseAwsResource
	Client       s3controliface.S3ControlAPI
	Region       string
	AccessPoints []string
	AccountID    *string
}

func (ap *S3ObjectLambdaAccessPoint) Init(session *session.Session) {
	ap.Client = s3control.New(session)
}

func (ap *S3ObjectLambdaAccessPoint) ResourceName() string {
	return "s3-olap"
}

func (ap *S3ObjectLambdaAccessPoint) ResourceIdentifiers() []string {
	return ap.AccessPoints
}

func (ap *S3ObjectLambdaAccessPoint) MaxBatchSize() int {
	return 5
}

func (ap *S3ObjectLambdaAccessPoint) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.S3ObjectLambdaAccessPoint
}

func (ap *S3ObjectLambdaAccessPoint) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ap.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ap.AccessPoints = awsgo.StringValueSlice(identifiers)
	return ap.AccessPoints, nil
}

func (ap *S3ObjectLambdaAccessPoint) Nuke(identifiers []string) error {
	if err := ap.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

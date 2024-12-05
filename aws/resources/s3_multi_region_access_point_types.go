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

type S3MultiRegionAccessPoint struct {
	BaseAwsResource
	Client       s3controliface.S3ControlAPI
	Region       string
	AccessPoints []string
	AccountID    *string
}

func (ap *S3MultiRegionAccessPoint) Init(session *session.Session) {
	ap.Client = s3control.New(session)
}

func (ap *S3MultiRegionAccessPoint) ResourceName() string {
	return "s3-mrap"
}

func (ap *S3MultiRegionAccessPoint) ResourceIdentifiers() []string {
	return ap.AccessPoints
}

func (ap *S3MultiRegionAccessPoint) MaxBatchSize() int {
	return 5
}

func (ap *S3MultiRegionAccessPoint) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.S3MultiRegionAccessPoint
}

func (ap *S3MultiRegionAccessPoint) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ap.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ap.AccessPoints = awsgo.StringValueSlice(identifiers)
	return ap.AccessPoints, nil
}

func (ap *S3MultiRegionAccessPoint) Nuke(identifiers []string) error {
	if err := ap.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

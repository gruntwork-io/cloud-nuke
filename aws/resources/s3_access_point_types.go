package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type S3ControlAccessPointAPI interface {
	ListAccessPoints(ctx context.Context, params *s3control.ListAccessPointsInput, optFns ...func(*s3control.Options)) (*s3control.ListAccessPointsOutput, error)
	DeleteAccessPoint(ctx context.Context, params *s3control.DeleteAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.DeleteAccessPointOutput, error)
}
type S3AccessPoint struct {
	BaseAwsResource
	Client       S3ControlAccessPointAPI
	Region       string
	AccessPoints []string
	AccountID    *string
}

func (ap *S3AccessPoint) Init(cfg aws.Config) {
	ap.Client = s3control.NewFromConfig(cfg)
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

	ap.AccessPoints = aws.ToStringSlice(identifiers)
	return ap.AccessPoints, nil
}

func (ap *S3AccessPoint) Nuke(identifiers []string) error {
	if err := ap.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

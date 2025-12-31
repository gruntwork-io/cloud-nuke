package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type S3ControlMultiRegionAPI interface {
	ListMultiRegionAccessPoints(ctx context.Context, params *s3control.ListMultiRegionAccessPointsInput, optFns ...func(*s3control.Options)) (*s3control.ListMultiRegionAccessPointsOutput, error)
	DeleteMultiRegionAccessPoint(ctx context.Context, params *s3control.DeleteMultiRegionAccessPointInput, optFns ...func(*s3control.Options)) (*s3control.DeleteMultiRegionAccessPointOutput, error)
}
type S3MultiRegionAccessPoint struct {
	BaseAwsResource
	Client       S3ControlMultiRegionAPI
	Region       string
	AccessPoints []string
	AccountID    *string
}

func (ap *S3MultiRegionAccessPoint) Init(cfg aws.Config) {
	ap.Client = s3control.NewFromConfig(cfg)
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

	ap.AccessPoints = aws.ToStringSlice(identifiers)
	return ap.AccessPoints, nil
}

func (ap *S3MultiRegionAccessPoint) Nuke(ctx context.Context, identifiers []string) error {
	if err := ap.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

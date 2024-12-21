package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type S3ControlAPI interface {
	ListAccessPointsForObjectLambda(context.Context, *s3control.ListAccessPointsForObjectLambdaInput, ...func(*s3control.Options)) (*s3control.ListAccessPointsForObjectLambdaOutput, error)
	DeleteAccessPointForObjectLambda(context.Context, *s3control.DeleteAccessPointForObjectLambdaInput, ...func(*s3control.Options)) (*s3control.DeleteAccessPointForObjectLambdaOutput, error)
}
type S3ObjectLambdaAccessPoint struct {
	BaseAwsResource
	Client       S3ControlAPI
	Region       string
	AccessPoints []string
	AccountID    *string
}

func (ap *S3ObjectLambdaAccessPoint) InitV2(cfg aws.Config) {
	ap.Client = s3control.NewFromConfig(cfg)
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

	ap.AccessPoints = aws.ToStringSlice(identifiers)
	return ap.AccessPoints, nil
}

func (ap *S3ObjectLambdaAccessPoint) Nuke(identifiers []string) error {
	if err := ap.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

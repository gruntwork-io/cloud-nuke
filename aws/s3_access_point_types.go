package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// S3AccessPoints - represents all S3AccessPoints that should be deleted.
type S3AccessPoints struct {
	AccessPointNames []string
}

// ResourceName - the simple name of the aws resource
func (s3ap S3AccessPoints) ResourceName() string {
	return "s3-ap"
}

// ResourceIdentifiers - the name of s3 access points
func (s3ap S3AccessPoints) ResourceIdentifiers() []string {
	return s3ap.AccessPointNames
}

func (s3ap S3AccessPoints) MaxBatchSize() int {
	return 99
}

// Nuke - nuke 'em all!!!
func (s3ap S3AccessPoints) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllS3AccessPoints(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// S3ObjectLambdaAccessPoints - represents all S3ObjectLambdaAccessPoints that should be deleted.
type S3ObjectLambdaAccessPoints struct {
	ObjectLambdaAccessPoints []string
}

// ResourceName - the simple name of the aws resource
func (s3olap S3ObjectLambdaAccessPoints) ResourceName() string {
	return "s3-olap"
}

// ResourceIdentifiers - the name of s3 object lambda access points
func (s3olap S3ObjectLambdaAccessPoints) ResourceIdentifiers() []string {
	return s3olap.ObjectLambdaAccessPoints
}

func (s3olap S3ObjectLambdaAccessPoints) MaxBatchSize() int {
	return 99
}

// Nuke - nuke 'em all!!!
func (s3olap S3ObjectLambdaAccessPoints) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllS3ObjectLambdaAccessPoints(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// S3MultiRegionAccessPoints - represents all S3MultiRegionAccessPoints that should be deleted.
type S3MultiRegionAccessPoints struct {
	MultiRegionAccessPoints []string
}

// ResourceName - the simple name of the aws resource
func (s3mrap S3MultiRegionAccessPoints) ResourceName() string {
	return "s3-mrap"
}

// ResourceIdentifiers - the name of s3 multi region access points
func (s3mrap S3MultiRegionAccessPoints) ResourceIdentifiers() []string {
	return s3mrap.MultiRegionAccessPoints
}

func (s3mrap S3MultiRegionAccessPoints) MaxBatchSize() int {
	return 99
}

// Nuke - nuke 'em all!!!
func (s3mrap S3MultiRegionAccessPoints) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllS3MultiRegionAccessPoints(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type S3MultiRegionAccessPointDeleteError struct {
	name string
}

func (e S3MultiRegionAccessPointDeleteError) Error() string {
	return "S3 Multi Region Access Point:" + e.name + "was not deleted"
}

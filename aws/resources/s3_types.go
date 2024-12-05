package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// S3Buckets - represents all S3 Buckets
type S3Buckets struct {
	BaseAwsResource
	Client s3iface.S3API
	Region string
	Names  []string
}

func (bucket *S3Buckets) Init(session *session.Session) {
	bucket.Client = s3.New(session)
}

// ResourceName - the simple name of the aws resource
func (bucket *S3Buckets) ResourceName() string {
	return "s3"
}

// MaxBatchSize decides how many S3 buckets to delete in one call.
func (bucket *S3Buckets) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 500
}

// MaxConcurrentGetSize decides how many S3 buckets to fetch in one call.
func (bucket *S3Buckets) MaxConcurrentGetSize() int {
	// To speed up bucket fetch part.
	return 100
}

// ObjectMaxBatchSize decides how many unique objects of an S3 bucket (object + version = unique object) to delete in one call.
func (bucket *S3Buckets) ObjectMaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 1000
}

// ResourceIdentifiers - The names of the S3 buckets
func (bucket *S3Buckets) ResourceIdentifiers() []string {
	return bucket.Names
}

// To get the resource configuration
func (bucket *S3Buckets) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.S3
}

func (bucket *S3Buckets) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := bucket.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	bucket.Names = aws.StringValueSlice(identifiers)
	return bucket.Names, nil
}

// Nuke - nuke 'em all!!!
func (bucket *S3Buckets) Nuke(identifiers []string) error {
	delCount, err := bucket.nukeAll(aws.StringSlice(identifiers))

	totalCount := len(identifiers)
	if delCount > 0 {
		logging.Debugf("[OK] - %d/%d - S3 bucket(s) deleted in %s", delCount, totalCount, bucket.Region)
	}
	if delCount != totalCount {
		logging.Debugf("[Failed] - %d/%d - S3 bucket(s) failed deletion in %s", totalCount-delCount, totalCount, bucket.Region)
	}

	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

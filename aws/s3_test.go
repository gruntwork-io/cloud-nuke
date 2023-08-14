package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedS3Buckets struct {
	s3iface.S3API
	ListBucketsOutput             s3.ListBucketsOutput
	GetBucketLocationOutput       s3.GetBucketLocationOutput
	GetBucketTaggingOutput        s3.GetBucketTaggingOutput
	GetBucketVersioningOutput     s3.GetBucketVersioningOutput
	ListObjectVersionsPagesOutput s3.ListObjectVersionsOutput
	DeleteObjectsOutput           s3.DeleteObjectsOutput
	DeleteBucketPolicyOutput      s3.DeleteBucketPolicyOutput
	DeleteBucketOutput            s3.DeleteBucketOutput
}

func (m mockedS3Buckets) ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	return &m.ListBucketsOutput, nil
}

func (m mockedS3Buckets) GetBucketLocation(*s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	return &m.GetBucketLocationOutput, nil
}

func (m mockedS3Buckets) GetBucketTagging(*s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error) {
	return &m.GetBucketTaggingOutput, nil
}

func (m mockedS3Buckets) WaitUntilBucketNotExists(*s3.HeadBucketInput) error {
	return nil
}

func (m mockedS3Buckets) GetBucketVersioning(*s3.GetBucketVersioningInput) (*s3.GetBucketVersioningOutput, error) {
	return &m.GetBucketVersioningOutput, nil
}

func (m mockedS3Buckets) ListObjectVersionsPages(input *s3.ListObjectVersionsInput, fn func(*s3.ListObjectVersionsOutput, bool) bool) error {
	fn(&m.ListObjectVersionsPagesOutput, true)
	return nil
}

func (m mockedS3Buckets) DeleteObjects(*s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {
	return &m.DeleteObjectsOutput, nil
}

func (m mockedS3Buckets) DeleteBucketPolicy(*s3.DeleteBucketPolicyInput) (*s3.DeleteBucketPolicyOutput, error) {
	return &m.DeleteBucketPolicyOutput, nil
}

func (m mockedS3Buckets) DeleteBucket(*s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return &m.DeleteBucketOutput, nil
}

func TestS3Bucket_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-bucket-1"
	testName2 := "test-bucket-2"
	now := time.Now()
	sb := S3Buckets{
		Client: mockedS3Buckets{
			ListBucketsOutput: s3.ListBucketsOutput{
				Buckets: []*s3.Bucket{
					{
						Name:         aws.String(testName1),
						CreationDate: aws.Time(now),
					},
					{
						Name:         aws.String(testName2),
						CreationDate: aws.Time(now.Add(1)),
					},
				},
			},
			GetBucketLocationOutput: s3.GetBucketLocationOutput{
				LocationConstraint: aws.String("us-east-1"),
			},
			GetBucketTaggingOutput: s3.GetBucketTaggingOutput{
				TagSet: []*s3.Tag{},
			},
		},
		Region: "us-east-1",
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := sb.getAll(config.Config{
				S3: tc.configObj,
			})
			require.NoError(t, err)

			require.Equal(t, len(tc.expected), len(names))
			for _, name := range names {
				require.Contains(t, tc.expected, *name)
			}
		})
	}
}

func TestS3Bucket_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	sb := S3Buckets{
		Client: mockedS3Buckets{
			GetBucketVersioningOutput: s3.GetBucketVersioningOutput{
				Status: aws.String("Enabled"),
			},
			ListObjectVersionsPagesOutput: s3.ListObjectVersionsOutput{
				Versions: []*s3.ObjectVersion{
					{
						Key:       aws.String("test-key"),
						VersionId: aws.String("test-version-id"),
					},
				},
				DeleteMarkers: []*s3.DeleteMarkerEntry{
					{
						Key:       aws.String("test-key"),
						VersionId: aws.String("test-version-id"),
					},
				},
			},
			DeleteObjectsOutput:      s3.DeleteObjectsOutput{},
			DeleteBucketPolicyOutput: s3.DeleteBucketPolicyOutput{},
			DeleteBucketOutput:       s3.DeleteBucketOutput{},
		},
	}

	count, err := sb.nukeAll([]*string{aws.String("test-bucket")})
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedS3Buckets struct {
	S3API
	ListBucketsOutput           s3.ListBucketsOutput
	GetBucketLocationOutput     s3.GetBucketLocationOutput
	GetBucketTaggingOutput      s3.GetBucketTaggingOutput
	GetBucketVersioningOutput   s3.GetBucketVersioningOutput
	ListObjectVersionsOutput    s3.ListObjectVersionsOutput
	DeleteObjectsOutput         s3.DeleteObjectsOutput
	DeleteBucketPolicyOutput    s3.DeleteBucketPolicyOutput
	DeleteBucketOutput          s3.DeleteBucketOutput
	DeleteBucketLifecycleOutput s3.DeleteBucketLifecycleOutput
	ListObjectsV2Output         s3.ListObjectsV2Output
	HeadBucketOutput            s3.HeadBucketOutput
}

func (m mockedS3Buckets) ListBuckets(context.Context, *s3.ListBucketsInput, ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return &m.ListBucketsOutput, nil
}

func (m mockedS3Buckets) GetBucketLocation(context.Context, *s3.GetBucketLocationInput, ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	return &m.GetBucketLocationOutput, nil
}

func (m mockedS3Buckets) GetBucketTagging(context.Context, *s3.GetBucketTaggingInput, ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	return &m.GetBucketTaggingOutput, nil
}

func (m mockedS3Buckets) GetBucketVersioning(context.Context, *s3.GetBucketVersioningInput, ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
	return &m.GetBucketVersioningOutput, nil
}

func (m mockedS3Buckets) DeleteObjects(context.Context, *s3.DeleteObjectsInput, ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	return &m.DeleteObjectsOutput, nil
}

func (m mockedS3Buckets) DeleteBucketPolicy(context.Context, *s3.DeleteBucketPolicyInput, ...func(*s3.Options)) (*s3.DeleteBucketPolicyOutput, error) {
	return &m.DeleteBucketPolicyOutput, nil
}

func (m mockedS3Buckets) DeleteBucketLifecycle(ctx context.Context, params *s3.DeleteBucketLifecycleInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketLifecycleOutput, error) {
	return &m.DeleteBucketLifecycleOutput, nil
}

func (m mockedS3Buckets) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	return &m.DeleteBucketOutput, nil
}
func (m mockedS3Buckets) ListObjectVersions(ctx context.Context, params *s3.ListObjectVersionsInput, optFns ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {
	return &m.ListObjectVersionsOutput, nil
}
func (m mockedS3Buckets) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return &m.ListObjectsV2Output, nil
}
func (m mockedS3Buckets) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return &m.HeadBucketOutput, &types.NotFound{}
}
func (m mockedS3Buckets) WaitForOutput(ctx context.Context, params *s3.HeadBucketInput, maxWaitDur time.Duration, optFns ...func(*s3.BucketNotExistsWaiterOptions)) (*s3.HeadBucketOutput, error) {
	return nil, nil
}

func TestS3Bucket_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-bucket-1"
	testName2 := "test-bucket-2"
	now := time.Now()
	sb := S3Buckets{
		Client: mockedS3Buckets{
			ListBucketsOutput: s3.ListBucketsOutput{
				Buckets: []types.Bucket{
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
				LocationConstraint: "us-east-1",
			},
			GetBucketTaggingOutput: s3.GetBucketTaggingOutput{
				TagSet: []types.Tag{},
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
			names, err := sb.getAll(context.Background(), config.Config{
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

	t.Parallel()

	sb := S3Buckets{
		Client: mockedS3Buckets{
			GetBucketVersioningOutput: s3.GetBucketVersioningOutput{
				Status: ("Enabled"),
			},
			ListObjectVersionsOutput: s3.ListObjectVersionsOutput{
				Versions: []types.ObjectVersion{
					{
						Key:       aws.String("test-key"),
						VersionId: aws.String("test-version-id"),
					},
				},
				DeleteMarkers: []types.DeleteMarkerEntry{
					{
						Key:       aws.String("test-key"),
						VersionId: aws.String("test-version-id"),
					},
				},
			},
			DeleteObjectsOutput:      s3.DeleteObjectsOutput{},
			DeleteBucketPolicyOutput: s3.DeleteBucketPolicyOutput{},
			DeleteBucketOutput:       s3.DeleteBucketOutput{},
			HeadBucketOutput:         s3.HeadBucketOutput{},
		},
	}
	sb.Context = context.Background()

	count, err := sb.nukeAll([]*string{aws.String("test-bucket")})
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

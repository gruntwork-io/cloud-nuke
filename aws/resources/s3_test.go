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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedS3Buckets struct {
	S3API
	ListBucketsOutput           s3.ListBucketsOutput
	GetBucketLocationOutput     s3.GetBucketLocationOutput
	GetBucketTaggingOutput      s3.GetBucketTaggingOutput
	GetBucketTaggingError       error
	ListObjectVersionsOutput    s3.ListObjectVersionsOutput
	ListObjectsV2Output         s3.ListObjectsV2Output
	DeleteObjectsOutput         s3.DeleteObjectsOutput
	DeleteBucketPolicyOutput    s3.DeleteBucketPolicyOutput
	DeleteBucketLifecycleOutput s3.DeleteBucketLifecycleOutput
	DeleteBucketOutput          s3.DeleteBucketOutput
	HeadBucketOutput            s3.HeadBucketOutput
}

func (m mockedS3Buckets) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return &m.ListBucketsOutput, nil
}

func (m mockedS3Buckets) GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	return &m.GetBucketLocationOutput, nil
}

func (m mockedS3Buckets) GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	if m.GetBucketTaggingError != nil {
		return nil, m.GetBucketTaggingError
	}
	return &m.GetBucketTaggingOutput, nil
}

func (m mockedS3Buckets) ListObjectVersions(ctx context.Context, params *s3.ListObjectVersionsInput, optFns ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {
	return &m.ListObjectVersionsOutput, nil
}

func (m mockedS3Buckets) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return &m.ListObjectsV2Output, nil
}

func (m mockedS3Buckets) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	return &m.DeleteObjectsOutput, nil
}

func (m mockedS3Buckets) DeleteBucketPolicy(ctx context.Context, params *s3.DeleteBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketPolicyOutput, error) {
	return &m.DeleteBucketPolicyOutput, nil
}

func (m mockedS3Buckets) DeleteBucketLifecycle(ctx context.Context, params *s3.DeleteBucketLifecycleInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketLifecycleOutput, error) {
	return &m.DeleteBucketLifecycleOutput, nil
}

func (m mockedS3Buckets) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	return &m.DeleteBucketOutput, nil
}

func (m mockedS3Buckets) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return &m.HeadBucketOutput, &types.NotFound{}
}

func TestS3Buckets_List(t *testing.T) {
	t.Parallel()

	testName1 := "test-bucket-1"
	testName2 := "test-bucket-2"
	now := time.Now()

	mockClient := mockedS3Buckets{
		ListBucketsOutput: s3.ListBucketsOutput{
			Buckets: []types.Bucket{
				{Name: aws.String(testName1), CreationDate: aws.Time(now)},
				{Name: aws.String(testName2), CreationDate: aws.Time(now.Add(time.Hour))},
			},
		},
		GetBucketLocationOutput: s3.GetBucketLocationOutput{
			LocationConstraint: "us-east-1",
		},
		GetBucketTaggingOutput: s3.GetBucketTaggingOutput{
			TagSet: []types.Tag{},
		},
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
					}},
				},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listS3Buckets(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestS3Buckets_GetBucketRegion(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		locationConstraint types.BucketLocationConstraint
		expectedRegion     string
	}{
		"us-east-1 returns empty": {
			locationConstraint: "",
			expectedRegion:     "us-east-1",
		},
		"us-west-2": {
			locationConstraint: "us-west-2",
			expectedRegion:     "us-west-2",
		},
		"eu-west-1": {
			locationConstraint: "eu-west-1",
			expectedRegion:     "eu-west-1",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := mockedS3Buckets{
				GetBucketLocationOutput: s3.GetBucketLocationOutput{
					LocationConstraint: tc.locationConstraint,
				},
			}

			region, err := getBucketRegion(context.Background(), mockClient, "test-bucket")
			require.NoError(t, err)
			require.Equal(t, tc.expectedRegion, region)
		})
	}
}

func TestS3Buckets_EmptyBucket(t *testing.T) {
	t.Parallel()

	mockClient := mockedS3Buckets{
		ListObjectVersionsOutput: s3.ListObjectVersionsOutput{
			Versions: []types.ObjectVersion{
				{Key: aws.String("test-key"), VersionId: aws.String("test-version-id")},
			},
			DeleteMarkers: []types.DeleteMarkerEntry{
				{Key: aws.String("test-key"), VersionId: aws.String("test-marker-id")},
			},
			IsTruncated: aws.Bool(false),
		},
		ListObjectsV2Output: s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("test-object")},
			},
		},
		DeleteObjectsOutput: s3.DeleteObjectsOutput{},
	}

	err := emptyBucket(context.Background(), mockClient, aws.String("test-bucket"))
	require.NoError(t, err)
}

func TestS3Buckets_DeleteBucketSteps(t *testing.T) {
	t.Parallel()

	mockClient := mockedS3Buckets{
		DeleteBucketPolicyOutput:    s3.DeleteBucketPolicyOutput{},
		DeleteBucketLifecycleOutput: s3.DeleteBucketLifecycleOutput{},
		DeleteBucketOutput:          s3.DeleteBucketOutput{},
		HeadBucketOutput:            s3.HeadBucketOutput{},
	}

	// Test individual deletion steps
	t.Run("deleteBucketPolicy", func(t *testing.T) {
		err := deleteBucketPolicy(context.Background(), mockClient, aws.String("test-bucket"))
		require.NoError(t, err)
	})

	t.Run("deleteBucketLifecycle", func(t *testing.T) {
		err := deleteBucketLifecycle(context.Background(), mockClient, aws.String("test-bucket"))
		require.NoError(t, err)
	})
}

func TestS3Buckets_DeleteObjectVersions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		versions []types.ObjectVersion
	}{
		"empty versions": {
			versions: []types.ObjectVersion{},
		},
		"single version": {
			versions: []types.ObjectVersion{
				{Key: aws.String("key1"), VersionId: aws.String("v1")},
			},
		},
		"multiple versions": {
			versions: []types.ObjectVersion{
				{Key: aws.String("key1"), VersionId: aws.String("v1")},
				{Key: aws.String("key2"), VersionId: aws.String("v2")},
				{Key: aws.String("key3"), VersionId: aws.String("v3")},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := mockedS3Buckets{
				DeleteObjectsOutput: s3.DeleteObjectsOutput{},
			}

			err := deleteObjectVersions(context.Background(), mockClient, aws.String("test-bucket"), tc.versions)
			require.NoError(t, err)
		})
	}
}

func TestS3Buckets_DeleteDeletionMarkers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		markers []types.DeleteMarkerEntry
	}{
		"empty markers": {
			markers: []types.DeleteMarkerEntry{},
		},
		"single marker": {
			markers: []types.DeleteMarkerEntry{
				{Key: aws.String("key1"), VersionId: aws.String("m1")},
			},
		},
		"multiple markers": {
			markers: []types.DeleteMarkerEntry{
				{Key: aws.String("key1"), VersionId: aws.String("m1")},
				{Key: aws.String("key2"), VersionId: aws.String("m2")},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := mockedS3Buckets{
				DeleteObjectsOutput: s3.DeleteObjectsOutput{},
			}

			err := deleteDeletionMarkers(context.Background(), mockClient, aws.String("test-bucket"), tc.markers)
			require.NoError(t, err)
		})
	}
}

package resources

import (
	"context"
	goerr "errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	// AwsResourceExclusionTagKey is the tag key used to exclude resources from deletion.
	AwsResourceExclusionTagKey = "cloud-nuke-excluded"

	// s3BucketDeletionRetries is the maximum number of retries for waiting on bucket deletion.
	s3BucketDeletionRetries = 3

	// s3BucketWaitDuration is the duration to wait for bucket deletion propagation.
	// S3 buckets take longer to propagate than other resources.
	s3BucketWaitDuration = 100 * time.Second
)

// S3API defines the interface for S3 operations.
type S3API interface {
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
	GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error)
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	ListObjectVersions(ctx context.Context, params *s3.ListObjectVersionsInput, optFns ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
	DeleteBucketPolicy(ctx context.Context, params *s3.DeleteBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketPolicyOutput, error)
	DeleteBucketLifecycle(ctx context.Context, params *s3.DeleteBucketLifecycleInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketLifecycleOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// s3BucketInfo holds information about an S3 bucket during discovery.
type s3BucketInfo struct {
	Name          string
	CreationDate  time.Time
	Tags          map[string]string
	Error         error
	IsValid       bool
	InvalidReason string
}

// NewS3Buckets creates a new S3Buckets resource using the generic resource pattern.
func NewS3Buckets() AwsResource {
	return NewAwsResource(&resource.Resource[S3API]{
		ResourceTypeName: "s3",
		BatchSize:        500,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[S3API], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = s3.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.S3
		},
		Lister: listS3Buckets,
		Nuker:  nukeS3Buckets,
	})
}

// s3RegionOption directs an S3 request at the given region. S3 is a global
// resource, so the shared client targets the global region; bucket-scoped calls
// (tagging, deletes, etc.) must be pointed at the bucket's own region or they
// return a 301 PermanentRedirect. Callers resolve the region via GetBucketLocation.
func s3RegionOption(region string) func(*s3.Options) {
	return func(o *s3.Options) {
		o.Region = region
	}
}

// listS3Buckets retrieves all S3 buckets that match the config filters.
func listS3Buckets(ctx context.Context, client S3API, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	if len(output.Buckets) == 0 {
		return nil, nil
	}

	// Process buckets concurrently in batches
	const batchSize = 100
	var allNames []*string

	for batchStart := 0; batchStart < len(output.Buckets); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(output.Buckets) {
			batchEnd = len(output.Buckets)
		}

		targetBuckets := output.Buckets[batchStart:batchEnd]
		names := getBucketNamesForBatch(ctx, client, scope, targetBuckets, cfg)
		allNames = append(allNames, names...)
	}

	return allNames, nil
}

// getBucketNamesForBatch processes a batch of buckets concurrently and returns valid bucket names.
func getBucketNamesForBatch(ctx context.Context, client S3API, scope resource.Scope, buckets []types.Bucket, cfg config.ResourceType) []*string {
	var names []*string
	resultCh := make(chan *s3BucketInfo, len(buckets))
	var wg sync.WaitGroup

	for _, bucket := range buckets {
		wg.Add(1)
		go func(b types.Bucket) {
			defer wg.Done()
			info := getBucketInfo(ctx, client, scope, b, cfg)
			resultCh <- info
		}(bucket)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for info := range resultCh {
		if info.Error != nil {
			logging.Debugf("Skipping bucket %s: %v", info.Name, info.Error)
			continue
		}
		if !info.IsValid {
			logging.Debugf("Skipping bucket %s: %s", info.Name, info.InvalidReason)
			continue
		}
		names = append(names, aws.String(info.Name))
	}

	return names
}

// getBucketInfo retrieves information about a single bucket.
func getBucketInfo(ctx context.Context, client S3API, scope resource.Scope, bucket types.Bucket, cfg config.ResourceType) *s3BucketInfo {
	info := &s3BucketInfo{
		Name:         aws.ToString(bucket.Name),
		CreationDate: aws.ToTime(bucket.CreationDate),
	}

	// Get bucket region
	region, err := getBucketRegion(ctx, client, info.Name)
	if err != nil {
		info.Error = err
		return info
	}

	// S3 is global but buckets exist in specific regions - we still need to filter by target region
	// to ensure we only process buckets in the region being queried
	if region != scope.Region && scope.Region != "global" {
		info.InvalidReason = "not in target region"
		return info
	}

	// Tag the bucket in its own region to avoid a 301 PermanentRedirect.
	tags, err := getBucketTags(ctx, client, info.Name, s3RegionOption(region))
	if err != nil {
		info.Error = err
		return info
	}
	info.Tags = tags

	// Apply config filters
	if !cfg.ShouldInclude(config.ResourceValue{
		Time: &info.CreationDate,
		Name: &info.Name,
		Tags: tags,
	}) {
		info.InvalidReason = "filtered by config"
		return info
	}

	info.IsValid = true
	return info
}

// getBucketRegion returns the region for an S3 bucket.
func getBucketRegion(ctx context.Context, client S3API, bucketName string) (string, error) {
	result, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	// GetBucketLocation returns empty string for us-east-1
	// https://github.com/aws/aws-sdk-go/issues/1687
	if result.LocationConstraint == "" {
		return "us-east-1", nil
	}
	return string(result.LocationConstraint), nil
}

// getBucketTags returns the tags for an S3 bucket.
func getBucketTags(ctx context.Context, client S3API, bucketName string, opts ...func(*s3.Options)) (map[string]string, error) {
	result, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	}, opts...)
	if err != nil {
		var apiErr *smithy.OperationError
		if goerr.As(err, &apiErr) {
			if strings.Contains(apiErr.Error(), "NoSuchTagSet: The TagSet does not exist") {
				return nil, nil
			}
		}
		return nil, errors.WithStackTrace(err)
	}
	return util.ConvertS3TypesTagsToMap(result.TagSet), nil
}

// nukeS3Buckets runs the deletion steps sequentially per bucket, stopping a
// bucket at its first failing step. Unlike resource.MultiStepDeleter it resolves
// each bucket's region up front and directs every step at that region, so
// buckets outside the global region are deleted instead of 301-redirected.
func nukeS3Buckets(ctx context.Context, client S3API, scope resource.Scope, resourceType string, identifiers []*string) []resource.NukeResult {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in %s", resourceType, scope)
		return nil
	}
	logging.Infof("Deleting %d %s in %s", len(identifiers), resourceType, scope)

	steps := []func(context.Context, S3API, *string, ...func(*s3.Options)) error{
		emptyBucket,
		deleteBucketPolicy,
		deleteBucketLifecycle,
		deleteBucketWithWait,
	}

	results := make([]resource.NukeResult, 0, len(identifiers))
	for _, id := range identifiers {
		name := aws.ToString(id)

		region, err := getBucketRegion(ctx, client, name)
		if err != nil {
			results = append(results, resource.NukeResult{Identifier: name, Error: errors.WithStackTrace(err)})
			continue
		}
		regionOpt := s3RegionOption(region)

		var stepErr error
		for i, step := range steps {
			if err := step(ctx, client, id, regionOpt); err != nil {
				stepErr = fmt.Errorf("step %d: %w", i+1, err)
				break
			}
		}

		results = append(results, resource.NukeResult{Identifier: name, Error: stepErr})
	}

	return results
}

// emptyBucket deletes all objects, versions, and deletion markers from a bucket.
func emptyBucket(ctx context.Context, client S3API, bucketName *string, opts ...func(*s3.Options)) error {
	logging.Debugf("Emptying bucket %s", aws.ToString(bucketName))

	// Delete all object versions and deletion markers
	if err := deleteAllVersionsAndMarkers(ctx, client, bucketName, opts...); err != nil {
		return errors.WithStackTrace(err)
	}

	// Delete any remaining unversioned objects
	if err := deleteAllObjects(ctx, client, bucketName, opts...); err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[OK] Emptied bucket %s", aws.ToString(bucketName))
	return nil
}

// deleteAllVersionsAndMarkers deletes all object versions and deletion markers.
func deleteAllVersionsAndMarkers(ctx context.Context, client S3API, bucketName *string, opts ...func(*s3.Options)) error {
	const maxKeys = 1000
	var keyMarker, versionIdMarker *string
	pageId := 1

	for {
		input := &s3.ListObjectVersionsInput{
			Bucket:  bucketName,
			MaxKeys: aws.Int32(maxKeys),
		}
		if keyMarker != nil {
			input.KeyMarker = keyMarker
		}
		if versionIdMarker != nil {
			input.VersionIdMarker = versionIdMarker
		}

		output, err := client.ListObjectVersions(ctx, input, opts...)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		// Delete object versions
		if len(output.Versions) > 0 {
			logging.Debugf("Deleting page %d of versions (%d) from bucket %s", pageId, len(output.Versions), aws.ToString(bucketName))
			if err := deleteObjectVersions(ctx, client, bucketName, output.Versions, opts...); err != nil {
				return errors.WithStackTrace(err)
			}
		}

		// Delete deletion markers
		if len(output.DeleteMarkers) > 0 {
			logging.Debugf("Deleting page %d of deletion markers (%d) from bucket %s", pageId, len(output.DeleteMarkers), aws.ToString(bucketName))
			if err := deleteDeletionMarkers(ctx, client, bucketName, output.DeleteMarkers, opts...); err != nil {
				return errors.WithStackTrace(err)
			}
		}

		if !aws.ToBool(output.IsTruncated) {
			break
		}

		keyMarker = output.NextKeyMarker
		versionIdMarker = output.NextVersionIdMarker
		pageId++
	}

	return nil
}

// deleteObjectVersions deletes a batch of object versions.
func deleteObjectVersions(ctx context.Context, client S3API, bucketName *string, versions []types.ObjectVersion, opts ...func(*s3.Options)) error {
	if len(versions) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, len(versions))
	for i, v := range versions {
		objects[i] = types.ObjectIdentifier{
			Key:       v.Key,
			VersionId: v.VersionId,
		}
	}

	_, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: bucketName,
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	}, opts...)
	return errors.WithStackTrace(err)
}

// deleteDeletionMarkers deletes a batch of deletion markers.
func deleteDeletionMarkers(ctx context.Context, client S3API, bucketName *string, markers []types.DeleteMarkerEntry, opts ...func(*s3.Options)) error {
	if len(markers) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, len(markers))
	for i, m := range markers {
		objects[i] = types.ObjectIdentifier{
			Key:       m.Key,
			VersionId: m.VersionId,
		}
	}

	_, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: bucketName,
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	}, opts...)
	return errors.WithStackTrace(err)
}

// deleteAllObjects deletes all remaining unversioned objects from a bucket.
func deleteAllObjects(ctx context.Context, client S3API, bucketName *string, opts ...func(*s3.Options)) error {
	const maxKeys = 1000
	pageId := 1

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:  bucketName,
		MaxKeys: aws.Int32(maxKeys),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx, opts...)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		if len(page.Contents) == 0 {
			continue
		}

		logging.Debugf("Deleting page %d of objects (%d) from bucket %s", pageId, len(page.Contents), aws.ToString(bucketName))

		objects := make([]types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			objects[i] = types.ObjectIdentifier{Key: obj.Key}
		}

		_, err = client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: bucketName,
			Delete: &types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		}, opts...)
		if err != nil {
			return errors.WithStackTrace(err)
		}
		pageId++
	}

	return nil
}

// deleteBucketPolicy deletes the bucket policy.
func deleteBucketPolicy(ctx context.Context, client S3API, bucketName *string, opts ...func(*s3.Options)) error {
	_, err := client.DeleteBucketPolicy(ctx, &s3.DeleteBucketPolicyInput{
		Bucket: bucketName,
	}, opts...)
	return errors.WithStackTrace(err)
}

// deleteBucketLifecycle deletes the bucket lifecycle configuration.
func deleteBucketLifecycle(ctx context.Context, client S3API, bucketName *string, opts ...func(*s3.Options)) error {
	_, err := client.DeleteBucketLifecycle(ctx, &s3.DeleteBucketLifecycleInput{
		Bucket: bucketName,
	}, opts...)
	return errors.WithStackTrace(err)
}

// deleteBucketWithWait deletes the bucket and waits for deletion confirmation.
func deleteBucketWithWait(ctx context.Context, client S3API, bucketName *string, opts ...func(*s3.Options)) error {
	_, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: bucketName,
	}, opts...)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return waitForBucketDeletion(ctx, client, aws.ToString(bucketName), opts...)
}

// waitForBucketDeletion waits for bucket deletion to propagate.
func waitForBucketDeletion(ctx context.Context, client S3API, bucketName string, opts ...func(*s3.Options)) error {
	waiter := s3.NewBucketNotExistsWaiter(client)

	// Forward the region options to the waiter's internal HeadBucket calls.
	waiterOpts := func(wo *s3.BucketNotExistsWaiterOptions) {
		wo.ClientOptions = append(wo.ClientOptions, opts...)
	}

	for i := 0; i < s3BucketDeletionRetries; i++ {
		logging.Debugf("Waiting for bucket %s deletion (attempt %d/%d)", bucketName, i+1, s3BucketDeletionRetries)

		err := waiter.Wait(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		}, s3BucketWaitDuration, waiterOpts)

		if err == nil {
			logging.Debugf("Bucket %s deletion confirmed", bucketName)
			return nil
		}

		if i < s3BucketDeletionRetries-1 {
			logging.Debugf("Retry waiting for bucket %s deletion: %v", bucketName, err)
		}
	}

	return nil // Don't fail if we can't confirm deletion - the bucket was deleted
}

package resources

import (
	"context"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
)

func TestGCSBuckets_ResourceName(t *testing.T) {
	gcs := &GCSBuckets{}
	assert.Equal(t, "gcs-bucket", gcs.ResourceName())
}

func TestGCSBuckets_MaxBatchSize(t *testing.T) {
	gcs := &GCSBuckets{}
	assert.Equal(t, 50, gcs.MaxBatchSize())
}

func TestGCSBuckets_GetAndSetResourceConfig(t *testing.T) {
	gcs := &GCSBuckets{}
	configObj := config.Config{}

	resourceConfig := gcs.GetAndSetResourceConfig(configObj)
	assert.Equal(t, "10m", resourceConfig.Timeout)
}

func TestGCSBuckets_Init(t *testing.T) {
	gcs := &GCSBuckets{}
	gcs.Init("test-project")

	assert.Equal(t, "test-project", gcs.ProjectID)
	assert.NotNil(t, gcs.Nukables)
}

func TestGCSBuckets_GetNukableStatus(t *testing.T) {
	gcs := &GCSBuckets{}
	gcs.Init("test-project")

	// Test getting status for non-existent resource
	err, exists := gcs.GetNukableStatus("non-existent")
	assert.False(t, exists)
	assert.Nil(t, err)

	// Test getting status for existing resource
	gcs.SetNukableStatus("test-bucket", assert.AnError)
	err, exists = gcs.GetNukableStatus("test-bucket")
	assert.True(t, exists)
	assert.Error(t, err)
}

func TestGCSBuckets_IsNukable(t *testing.T) {
	gcs := &GCSBuckets{}
	gcs.Init("test-project")

	// Test nukable status for non-existent resource (should be nukable by default)
	nukable, err := gcs.IsNukable("non-existent")
	assert.True(t, nukable)
	assert.Nil(t, err)

	// Test nukable status for resource with error
	gcs.SetNukableStatus("test-bucket", assert.AnError)
	nukable, err = gcs.IsNukable("test-bucket")
	assert.False(t, nukable)
	assert.Error(t, err)

	// Test nukable status for resource without error
	gcs.SetNukableStatus("test-bucket-2", nil)
	nukable, err = gcs.IsNukable("test-bucket-2")
	assert.True(t, nukable)
	assert.Nil(t, err)
}

func TestGCSBuckets_PrepareContext(t *testing.T) {
	gcs := &GCSBuckets{}
	gcs.Init("test-project")

	// Test with timeout
	resourceConfig := config.ResourceType{Timeout: "5s"}
	err := gcs.PrepareContext(context.Background(), resourceConfig)
	assert.NoError(t, err)
	assert.NotNil(t, gcs.Context)
	assert.True(t, gcs.HasCancelFunc(), "cancel function should exist when timeout is set")

	// Clean up
	gcs.CancelContext()

	// Test without timeout - create a new instance to test clean state
	gcs2 := &GCSBuckets{}
	resourceConfig = config.ResourceType{Timeout: ""}
	err = gcs2.PrepareContext(context.Background(), resourceConfig)
	assert.NoError(t, err)
	assert.NotNil(t, gcs2.Context)
	// When no timeout is specified, the context should be the same as the parent
	// and cancel should be nil (no timeout context created)
	assert.False(t, gcs2.HasCancelFunc(), "cancel function should not exist when timeout is not set")
}

func TestGCSBuckets_ResourceIdentifiers(t *testing.T) {
	gcs := &GCSBuckets{
		Names: []string{"bucket1", "bucket2", "bucket3"},
	}

	identifiers := gcs.ResourceIdentifiers()
	assert.Equal(t, []string{"bucket1", "bucket2", "bucket3"}, identifiers)
}

package resources

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/gruntwork-io/cloud-nuke/config"
)

// GCSBucketAPI represents the interface for GCS bucket operations
type GCSBucketAPI interface {
	Buckets(ctx context.Context, projectID string) *storage.BucketIterator
	Bucket(name string) *storage.BucketHandle
}

// GCSBuckets represents all GCS buckets in a project
type GCSBuckets struct {
	BaseGcpResource
	Client GCSBucketAPI
	Names  []string
}

func (b *GCSBuckets) Init(projectID string) {
	b.BaseGcpResource.Init(projectID)
	client, err := storage.NewClient(context.Background())
	if err != nil {
		// Handle error appropriately
		return
	}
	b.Client = client
}

func (b *GCSBuckets) ResourceName() string {
	return "gcs-bucket"
}

func (b *GCSBuckets) ResourceIdentifiers() []string {
	return b.Names
}

func (b *GCSBuckets) MaxBatchSize() int {
	// GCS has a relatively high API quota, but we'll be conservative
	return 50
}

func (b *GCSBuckets) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	// TODO: Add GCS configuration to config package
	return config.ResourceType{
		Timeout: "10m", // Default timeout
	}
}

func (b *GCSBuckets) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := b.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	b.Names = identifiers
	return b.Names, nil
}

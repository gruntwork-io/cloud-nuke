package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AwsResource is the single canonical interface for all AWS resources.
// The aws package references this type for container types and the resource registry.
type AwsResource interface {
	Init(cfg aws.Config)
	ResourceName() string
	ResourceIdentifiers() []string
	MaxBatchSize() int
	Nuke(ctx context.Context, identifiers []string) ([]resource.NukeResult, error)
	GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error)
	IsNukable(string) (bool, error)
	GetAndSetResourceConfig(config.Config) config.ResourceType
}

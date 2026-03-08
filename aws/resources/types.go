package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AwsResource is an interface that represents a single AWS resource.
// This interface is structurally identical to aws.AwsResource but defined here
// to avoid a circular import (aws imports resources). Go's structural typing
// ensures that types satisfying this interface also satisfy aws.AwsResource.
// This interface is satisfied by AwsResourceAdapter[C] which wraps resource.Resource[C].
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

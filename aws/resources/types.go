package resources

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AwsResource is the single canonical interface for all AWS resources.
// The aws package references this type for container types and the resource registry.
type AwsResource interface {
	resource.NukeableResource
	Init(cfg aws.Config)
}

package resources

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AwsResource is an interface that represents a single AWS resource.
// It embeds the provider-agnostic NukeableResource interface and adds AWS-specific Init.
// This interface is structurally identical to aws.AwsResource but defined here
// to avoid a circular import (aws imports resources). Go's structural typing
// ensures that types satisfying this interface also satisfy aws.AwsResource.
type AwsResource interface {
	resource.NukeableResource
	Init(cfg aws.Config)
}

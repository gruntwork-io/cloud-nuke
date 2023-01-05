package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2DedicatedHosts - represents all host allocation IDs
type EC2DedicatedHosts struct {
	HostIds []string
}

// ResourceName - the simple name of the aws resource
func (h EC2DedicatedHosts) ResourceName() string {
	return "ec2-dedicated-hosts"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (h EC2DedicatedHosts) ResourceIdentifiers() []string {
	return h.HostIds
}

func (h EC2DedicatedHosts) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (h EC2DedicatedHosts) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllEc2DedicatedHosts(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

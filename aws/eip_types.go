package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// EBSVolume - represents all ebs volumes
type ElasticIP struct {
	Client        ec2iface.EC2API
	Region        string
	AllocationIds []string
}

// ResourceName - the simple name of the aws resource
func (address ElasticIP) ResourceName() string {
	return "eip"
}

// ResourceIdentifiers - The instance ids of the eip addresses
func (address ElasticIP) ResourceIdentifiers() []string {
	return address.AllocationIds
}

func (address ElasticIP) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (address ElasticIP) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllEIPAddresses(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// EBSVolumes - represents all ebs volumes
type EIPAddresses struct {
	AllocationIds []string
}

// ResourceName - the simple name of the aws resource
func (address EIPAddresses) ResourceName() string {
	return "eip"
}

// ResourceIdentifiers - The instance ids of the eip addresses
func (address EIPAddresses) ResourceIdentifiers() []string {
	return address.AllocationIds
}

func (address EIPAddresses) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

// Nuke - nuke 'em all!!!
func (address EIPAddresses) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllEIPAddresses(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

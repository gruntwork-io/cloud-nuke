package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudformationStacks struct {
	StackIds []string
}

// ResourceName - the simple name of the aws resource
func (stack CloudformationStacks) ResourceName() string {
	return "cf-stack"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (stack CloudformationStacks) ResourceIdentifiers() []string {
	return stack.StackIds
}

func (stack CloudformationStacks) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

// Nuke - nuke 'em all!!!
func (stack CloudformationStacks) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllCloudformationStacks(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

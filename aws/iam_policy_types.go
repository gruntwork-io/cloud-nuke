package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMPolicies - represents all IAM Policies on the AWS account
type IAMPolicies struct {
	PolicyArns []string
}

// ResourceName - the simple name of the AWS resource
func (p IAMPolicies) ResourceName() string {
	return "iam-policy"
}

// ResourceIdentifiers - The IAM GroupNames
func (p IAMPolicies) ResourceIdentifiers() []string {
	return p.PolicyArns
}

// MaxBatchSize Tentative batch size to ensure AWS doesn't throttle
func (p IAMPolicies) MaxBatchSize() int {
	return 20
}

// Nuke - Destroy every group in this collection
func (p IAMPolicies) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllIamPolicies(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

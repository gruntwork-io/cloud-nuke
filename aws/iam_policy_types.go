package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMPolicies - represents all IAM Policies on the AWS account
type IAMPolicies struct {
	Client     iamiface.IAMAPI
	PolicyArns []string
}

// ResourceName - the simple name of the AWS resource
func (ip IAMPolicies) ResourceName() string {
	return "iam-policy"
}

// ResourceIdentifiers - The IAM GroupNames
func (ip IAMPolicies) ResourceIdentifiers() []string {
	return ip.PolicyArns
}

// MaxBatchSize Tentative batch size to ensure AWS doesn't throttle
func (ip IAMPolicies) MaxBatchSize() int {
	return 20
}

// Nuke - Destroy every group in this collection
func (ip IAMPolicies) Nuke(session *session.Session, identifiers []string) error {
	if err := ip.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

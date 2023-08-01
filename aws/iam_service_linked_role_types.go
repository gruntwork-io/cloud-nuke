package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMServiceLinkedRoles - represents all IAMServiceLinkedRoles on the AWS Account
type IAMServiceLinkedRoles struct {
	Client    iamiface.IAMAPI
	RoleNames []string
}

// ResourceName - the simple name of the aws resource
func (islr IAMServiceLinkedRoles) ResourceName() string {
	return "iam-service-linked-role"
}

// ResourceIdentifiers - The IAM UserNames
func (islr IAMServiceLinkedRoles) ResourceIdentifiers() []string {
	return islr.RoleNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (islr IAMServiceLinkedRoles) MaxBatchSize() int {
	return 49
}

// Nuke - nuke 'em all!!!
func (islr IAMServiceLinkedRoles) Nuke(session *session.Session, identifiers []string) error {
	if err := islr.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

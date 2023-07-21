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
func (r IAMServiceLinkedRoles) ResourceName() string {
	return "iam-service-linked-role"
}

// ResourceIdentifiers - The IAM UserNames
func (r IAMServiceLinkedRoles) ResourceIdentifiers() []string {
	return r.RoleNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (r IAMServiceLinkedRoles) MaxBatchSize() int {
	return 49
}

// Nuke - nuke 'em all!!!
func (r IAMServiceLinkedRoles) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllIamServiceLinkedRoles(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

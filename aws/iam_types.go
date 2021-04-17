package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// IAMUsers - represents all IAMUsers on the AWS Account
type IAMUsers struct {
	UserNames []string
}

// ResourceName - the simple name of the aws resource
func (u IAMUsers) ResourceName() string {
	return "iam"
}

// ResourceIdentifiers - The IAM UserNames
func (u IAMUsers) ResourceIdentifiers() []string {
	return u.UserNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (u IAMUsers) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (u IAMUsers) Nuke(session *session.Session, users []string) error {
	if err := nukeAllIamUsers(session, awsgo.StringSlice(users)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

package aws

import (
	"fmt"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
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

// IAMRoles - represents all IAMRoles in the AWS Account
type IAMRoles struct {
	RoleNames []string
}

// ResourceName - the simple name of the aws resource
func (r IAMRoles) ResourceName() string {
	return "iam-roles"
}

// ResourceIdentifiers - The IAM UserNames
func (r IAMRoles) ResourceIdentifiers() []string {
	return r.RoleNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (r IAMRoles) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (r IAMRoles) Nuke(session *session.Session, roles []string) error {
	if err := nukeAllIamRoles(session, awsgo.StringSlice(roles)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Custom error types

type ManagedIAMRoleErr struct {
	Name string
}

func (err ManagedIAMRoleErr) Error() string {
	return fmt.Sprintf("Could not nuke IAM Role %s because it is an AWS-managed role. Only AWS may modify / delete AWS-managed IAM roles", err.Name)
}

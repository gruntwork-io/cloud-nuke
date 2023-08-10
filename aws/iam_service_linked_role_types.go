package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
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

func (islr IAMServiceLinkedRoles) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := islr.getAll(configObj)
	if err != nil {
		return nil, err
	}

	islr.RoleNames = awsgo.StringValueSlice(identifiers)
	return islr.RoleNames, nil
}

// Nuke - nuke 'em all!!!
func (islr IAMServiceLinkedRoles) Nuke(identifiers []string) error {
	if err := islr.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

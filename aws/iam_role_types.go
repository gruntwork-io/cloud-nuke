package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMRoles - represents all IAMRoles on the AWS Account
type IAMRoles struct {
	Client    iamiface.IAMAPI
	RoleNames []string
}

// ResourceName - the simple name of the aws resource
func (ir IAMRoles) ResourceName() string {
	return "iam-role"
}

// ResourceIdentifiers - The IAM UserNames
func (ir IAMRoles) ResourceIdentifiers() []string {
	return ir.RoleNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (ir IAMRoles) MaxBatchSize() int {
	return 20
}

func (ir IAMRoles) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := ir.getAll(configObj)
	if err != nil {
		return nil, err
	}

	ir.RoleNames = awsgo.StringValueSlice(identifiers)
	return ir.RoleNames, nil
}

// Nuke - nuke 'em all!!!
func (ir IAMRoles) Nuke(identifiers []string) error {
	if err := ir.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

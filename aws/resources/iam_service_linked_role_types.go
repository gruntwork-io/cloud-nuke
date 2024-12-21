package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type IAMServiceLinkedRolesAPI interface {
	ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	DeleteServiceLinkedRole(ctx context.Context, params *iam.DeleteServiceLinkedRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteServiceLinkedRoleOutput, error)
	GetServiceLinkedRoleDeletionStatus(ctx context.Context, params *iam.GetServiceLinkedRoleDeletionStatusInput, optFns ...func(*iam.Options)) (*iam.GetServiceLinkedRoleDeletionStatusOutput, error)
}

// IAMServiceLinkedRoles - represents all IAMServiceLinkedRoles on the AWS Account
type IAMServiceLinkedRoles struct {
	BaseAwsResource
	Client    IAMServiceLinkedRolesAPI
	RoleNames []string
}

func (islr *IAMServiceLinkedRoles) Init(cfg aws.Config) {
	islr.Client = iam.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (islr *IAMServiceLinkedRoles) ResourceName() string {
	return "iam-service-linked-role"
}

// ResourceIdentifiers - The IAM UserNames
func (islr *IAMServiceLinkedRoles) ResourceIdentifiers() []string {
	return islr.RoleNames
}

// MaxBatchSize Tentative batch size to ensure AWS doesn't throttle
func (islr *IAMServiceLinkedRoles) MaxBatchSize() int {
	return 49
}

func (islr *IAMServiceLinkedRoles) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.IAMServiceLinkedRoles
}

func (islr *IAMServiceLinkedRoles) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := islr.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	islr.RoleNames = aws.ToStringSlice(identifiers)
	return islr.RoleNames, nil
}

// Nuke - nuke 'em all!!!
func (islr *IAMServiceLinkedRoles) Nuke(identifiers []string) error {
	if err := islr.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

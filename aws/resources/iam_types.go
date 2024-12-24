package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type IAMUsersAPI interface {
	ListAccessKeys(ctx context.Context, params *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
	ListGroupsForUser(ctx context.Context, params *iam.ListGroupsForUserInput, optFns ...func(*iam.Options)) (*iam.ListGroupsForUserOutput, error)
	ListUserPolicies(ctx context.Context, params *iam.ListUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListUserPoliciesOutput, error)
	ListUserTags(ctx context.Context, params *iam.ListUserTagsInput, optFns ...func(*iam.Options)) (*iam.ListUserTagsOutput, error)
	ListMFADevices(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error)
	ListSSHPublicKeys(ctx context.Context, params *iam.ListSSHPublicKeysInput, optFns ...func(*iam.Options)) (*iam.ListSSHPublicKeysOutput, error)
	ListServiceSpecificCredentials(ctx context.Context, params *iam.ListServiceSpecificCredentialsInput, optFns ...func(*iam.Options)) (*iam.ListServiceSpecificCredentialsOutput, error)
	ListSigningCertificates(ctx context.Context, params *iam.ListSigningCertificatesInput, optFns ...func(*iam.Options)) (*iam.ListSigningCertificatesOutput, error)
	ListUsers(ctx context.Context, params *iam.ListUsersInput, optFns ...func(*iam.Options)) (*iam.ListUsersOutput, error)
	ListAttachedUserPolicies(ctx context.Context, params *iam.ListAttachedUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedUserPoliciesOutput, error)
	DetachUserPolicy(ctx context.Context, params *iam.DetachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachUserPolicyOutput, error)
	DeleteLoginProfile(ctx context.Context, params *iam.DeleteLoginProfileInput, optFns ...func(*iam.Options)) (*iam.DeleteLoginProfileOutput, error)
	DeleteUserPolicy(ctx context.Context, params *iam.DeleteUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error)
	DeleteAccessKey(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
	DeleteSigningCertificate(ctx context.Context, params *iam.DeleteSigningCertificateInput, optFns ...func(*iam.Options)) (*iam.DeleteSigningCertificateOutput, error)
	DeleteSSHPublicKey(ctx context.Context, params *iam.DeleteSSHPublicKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteSSHPublicKeyOutput, error)
	DeleteServiceSpecificCredential(ctx context.Context, params *iam.DeleteServiceSpecificCredentialInput, optFns ...func(*iam.Options)) (*iam.DeleteServiceSpecificCredentialOutput, error)
	DeactivateMFADevice(ctx context.Context, params *iam.DeactivateMFADeviceInput, optFns ...func(*iam.Options)) (*iam.DeactivateMFADeviceOutput, error)
	DeleteVirtualMFADevice(ctx context.Context, params *iam.DeleteVirtualMFADeviceInput, optFns ...func(*iam.Options)) (*iam.DeleteVirtualMFADeviceOutput, error)
	DeleteUser(ctx context.Context, params *iam.DeleteUserInput, optFns ...func(*iam.Options)) (*iam.DeleteUserOutput, error)
	RemoveUserFromGroup(ctx context.Context, params *iam.RemoveUserFromGroupInput, optFns ...func(*iam.Options)) (*iam.RemoveUserFromGroupOutput, error)
}

// IAMUsers - represents all IAMUsers on the AWS Account
type IAMUsers struct {
	BaseAwsResource
	Client    IAMUsersAPI
	UserNames []string
}

func (iu *IAMUsers) Init(cfg aws.Config) {
	iu.Client = iam.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (iu *IAMUsers) ResourceName() string {
	return "iam"
}

// ResourceIdentifiers - The IAM UserNames
func (iu *IAMUsers) ResourceIdentifiers() []string {
	return iu.UserNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (iu *IAMUsers) MaxBatchSize() int {
	return 49
}

func (iu *IAMUsers) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.IAMUsers
}

func (iu *IAMUsers) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := iu.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	iu.UserNames = aws.ToStringSlice(identifiers)
	return iu.UserNames, nil
}

// Nuke - nuke 'em all!!!
func (iu *IAMUsers) Nuke(users []string) error {
	if err := iu.nukeAll(aws.StringSlice(users)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

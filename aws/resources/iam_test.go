package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedIAMUsers struct {
	IAMUsersAPI
	ListAccessKeysOutput                  iam.ListAccessKeysOutput
	ListGroupsForUserOutput               iam.ListGroupsForUserOutput
	ListUserPoliciesOutput                iam.ListUserPoliciesOutput
	ListUserTagsOutput                    iam.ListUserTagsOutput
	ListMFADevicesOutput                  iam.ListMFADevicesOutput
	ListSSHPublicKeysOutput               iam.ListSSHPublicKeysOutput
	ListServiceSpecificCredentialsOutput  iam.ListServiceSpecificCredentialsOutput
	ListSigningCertificatesOutput         iam.ListSigningCertificatesOutput
	ListUsersOutput                       iam.ListUsersOutput
	ListAttachedUserPoliciesOutput        iam.ListAttachedUserPoliciesOutput
	DetachUserPolicyOutput                iam.DetachUserPolicyOutput
	DeleteLoginProfileOutput              iam.DeleteLoginProfileOutput
	DeleteUserPolicyOutput                iam.DeleteUserPolicyOutput
	DeleteAccessKeyOutput                 iam.DeleteAccessKeyOutput
	DeleteSigningCertificateOutput        iam.DeleteSigningCertificateOutput
	DeleteSSHPublicKeyOutput              iam.DeleteSSHPublicKeyOutput
	DeleteServiceSpecificCredentialOutput iam.DeleteServiceSpecificCredentialOutput
	DeactivateMFADeviceOutput             iam.DeactivateMFADeviceOutput
	DeleteVirtualMFADeviceOutput          iam.DeleteVirtualMFADeviceOutput
	DeleteUserOutput                      iam.DeleteUserOutput
	RemoveUserFromGroupOutput             iam.RemoveUserFromGroupOutput
}

func (m mockedIAMUsers) ListAccessKeys(ctx context.Context, params *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
	return &m.ListAccessKeysOutput, nil
}

func (m mockedIAMUsers) ListGroupsForUser(ctx context.Context, params *iam.ListGroupsForUserInput, optFns ...func(*iam.Options)) (*iam.ListGroupsForUserOutput, error) {
	return &m.ListGroupsForUserOutput, nil
}

func (m mockedIAMUsers) ListUserPolicies(ctx context.Context, params *iam.ListUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListUserPoliciesOutput, error) {
	return &m.ListUserPoliciesOutput, nil
}

func (m mockedIAMUsers) ListUserTags(ctx context.Context, params *iam.ListUserTagsInput, optFns ...func(*iam.Options)) (*iam.ListUserTagsOutput, error) {
	return &m.ListUserTagsOutput, nil
}

func (m mockedIAMUsers) ListMFADevices(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error) {
	return &m.ListMFADevicesOutput, nil
}

func (m mockedIAMUsers) ListSSHPublicKeys(ctx context.Context, params *iam.ListSSHPublicKeysInput, optFns ...func(*iam.Options)) (*iam.ListSSHPublicKeysOutput, error) {
	return &m.ListSSHPublicKeysOutput, nil
}

func (m mockedIAMUsers) ListServiceSpecificCredentials(ctx context.Context, params *iam.ListServiceSpecificCredentialsInput, optFns ...func(*iam.Options)) (*iam.ListServiceSpecificCredentialsOutput, error) {
	return &m.ListServiceSpecificCredentialsOutput, nil
}

func (m mockedIAMUsers) ListSigningCertificates(ctx context.Context, params *iam.ListSigningCertificatesInput, optFns ...func(*iam.Options)) (*iam.ListSigningCertificatesOutput, error) {
	return &m.ListSigningCertificatesOutput, nil
}

func (m mockedIAMUsers) ListUsers(ctx context.Context, params *iam.ListUsersInput, optFns ...func(*iam.Options)) (*iam.ListUsersOutput, error) {
	return &m.ListUsersOutput, nil
}

func (m mockedIAMUsers) ListAttachedUserPolicies(ctx context.Context, params *iam.ListAttachedUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedUserPoliciesOutput, error) {
	return &m.ListAttachedUserPoliciesOutput, nil
}

func (m mockedIAMUsers) DetachUserPolicy(ctx context.Context, params *iam.DetachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachUserPolicyOutput, error) {
	return &m.DetachUserPolicyOutput, nil
}

func (m mockedIAMUsers) DeleteLoginProfile(ctx context.Context, params *iam.DeleteLoginProfileInput, optFns ...func(*iam.Options)) (*iam.DeleteLoginProfileOutput, error) {
	return &m.DeleteLoginProfileOutput, nil
}

func (m mockedIAMUsers) DeleteUserPolicy(ctx context.Context, params *iam.DeleteUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
	return &m.DeleteUserPolicyOutput, nil
}

func (m mockedIAMUsers) DeleteAccessKey(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
	return &m.DeleteAccessKeyOutput, nil
}

func (m mockedIAMUsers) DeleteSigningCertificate(ctx context.Context, params *iam.DeleteSigningCertificateInput, optFns ...func(*iam.Options)) (*iam.DeleteSigningCertificateOutput, error) {
	return &m.DeleteSigningCertificateOutput, nil
}

func (m mockedIAMUsers) DeleteSSHPublicKey(ctx context.Context, params *iam.DeleteSSHPublicKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteSSHPublicKeyOutput, error) {
	return &m.DeleteSSHPublicKeyOutput, nil
}

func (m mockedIAMUsers) DeleteServiceSpecificCredential(ctx context.Context, params *iam.DeleteServiceSpecificCredentialInput, optFns ...func(*iam.Options)) (*iam.DeleteServiceSpecificCredentialOutput, error) {
	return &m.DeleteServiceSpecificCredentialOutput, nil
}

func (m mockedIAMUsers) DeactivateMFADevice(ctx context.Context, params *iam.DeactivateMFADeviceInput, optFns ...func(*iam.Options)) (*iam.DeactivateMFADeviceOutput, error) {
	return &m.DeactivateMFADeviceOutput, nil
}

func (m mockedIAMUsers) DeleteVirtualMFADevice(ctx context.Context, params *iam.DeleteVirtualMFADeviceInput, optFns ...func(*iam.Options)) (*iam.DeleteVirtualMFADeviceOutput, error) {
	return &m.DeleteVirtualMFADeviceOutput, nil
}

func (m mockedIAMUsers) DeleteUser(ctx context.Context, params *iam.DeleteUserInput, optFns ...func(*iam.Options)) (*iam.DeleteUserOutput, error) {
	return &m.DeleteUserOutput, nil
}

func (m mockedIAMUsers) RemoveUserFromGroup(ctx context.Context, params *iam.RemoveUserFromGroupInput, optFns ...func(*iam.Options)) (*iam.RemoveUserFromGroupOutput, error) {
	return &m.RemoveUserFromGroupOutput, nil
}

func TestIAMUsers_GetAll(t *testing.T) {
	t.Parallel()
	now := time.Now()
	testName1 := "test-user1"
	testName2 := "test-user2"
	iu := IAMUsers{
		Client: mockedIAMUsers{
			ListUsersOutput: iam.ListUsersOutput{
				Users: []types.User{
					{
						UserName:   aws.String(testName1),
						CreateDate: aws.Time(now),
					},
					{
						UserName:   aws.String(testName2),
						CreateDate: aws.Time(now.Add(1)),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := iu.getAll(context.Background(), config.Config{
				IAMUsers: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestIAMUsers_NukeAll(t *testing.T) {
	t.Parallel()
	iu := IAMUsers{
		Client: mockedIAMUsers{
			ListAttachedUserPoliciesOutput: iam.ListAttachedUserPoliciesOutput{
				AttachedPolicies: []types.AttachedPolicy{
					{
						PolicyName: aws.String("test-policy"),
					},
				},
			},
			DetachUserPolicyOutput: iam.DetachUserPolicyOutput{},
			ListUserPoliciesOutput: iam.ListUserPoliciesOutput{
				PolicyNames: []string{
					"test-policy",
				},
			},
			DeleteUserPolicyOutput: iam.DeleteUserPolicyOutput{},
			ListGroupsForUserOutput: iam.ListGroupsForUserOutput{
				Groups: []types.Group{
					{
						GroupName: aws.String("test-group"),
					},
				},
			},
			RemoveUserFromGroupOutput: iam.RemoveUserFromGroupOutput{},
			DeleteLoginProfileOutput:  iam.DeleteLoginProfileOutput{},
			ListAccessKeysOutput: iam.ListAccessKeysOutput{
				AccessKeyMetadata: []types.AccessKeyMetadata{
					{
						AccessKeyId: aws.String("test-key"),
					},
				},
			},
			DeleteAccessKeyOutput: iam.DeleteAccessKeyOutput{},
			ListSigningCertificatesOutput: iam.ListSigningCertificatesOutput{
				Certificates: []types.SigningCertificate{
					{
						CertificateId: aws.String("test-certificate"),
					},
				},
			},
			DeleteSigningCertificateOutput: iam.DeleteSigningCertificateOutput{},
			ListSSHPublicKeysOutput: iam.ListSSHPublicKeysOutput{
				SSHPublicKeys: []types.SSHPublicKeyMetadata{
					{
						SSHPublicKeyId: aws.String("test-ssh-key"),
					},
				},
			},
			DeleteSSHPublicKeyOutput: iam.DeleteSSHPublicKeyOutput{},
			ListServiceSpecificCredentialsOutput: iam.ListServiceSpecificCredentialsOutput{
				ServiceSpecificCredentials: []types.ServiceSpecificCredentialMetadata{
					{
						ServiceSpecificCredentialId: aws.String("test-service-credential"),
					},
				},
			},
			DeleteServiceSpecificCredentialOutput: iam.DeleteServiceSpecificCredentialOutput{},
			ListMFADevicesOutput: iam.ListMFADevicesOutput{
				MFADevices: []types.MFADevice{
					{
						SerialNumber: aws.String("test-mfa-device"),
					},
				},
			},
			DeactivateMFADeviceOutput:    iam.DeactivateMFADeviceOutput{},
			DeleteVirtualMFADeviceOutput: iam.DeleteVirtualMFADeviceOutput{},
			DeleteUserOutput:             iam.DeleteUserOutput{},
		},
	}

	err := iu.nukeAll([]*string{aws.String("test-user")})
	require.NoError(t, err)
}

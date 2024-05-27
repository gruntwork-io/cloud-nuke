package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedIAMUsers struct {
	iamiface.IAMAPI
	ListUsersPagesOutput                  iam.ListUsersOutput
	ListAttachedUserPoliciesOutput        iam.ListAttachedUserPoliciesOutput
	DetachUserPolicyOutput                iam.DetachUserPolicyOutput
	ListUserPoliciesOutput                iam.ListUserPoliciesOutput
	DeleteUserPolicyOutput                iam.DeleteUserPolicyOutput
	ListGroupsForUserOutput               iam.ListGroupsForUserOutput
	RemoveUserFromGroupOutput             iam.RemoveUserFromGroupOutput
	DeleteLoginProfileOutput              iam.DeleteLoginProfileOutput
	ListAccessKeysOutput                  iam.ListAccessKeysOutput
	DeleteAccessKeyOutput                 iam.DeleteAccessKeyOutput
	ListSigningCertificatesOutput         iam.ListSigningCertificatesOutput
	DeleteSigningCertificateOutput        iam.DeleteSigningCertificateOutput
	ListSSHPublicKeysOutput               iam.ListSSHPublicKeysOutput
	DeleteSSHPublicKeyOutput              iam.DeleteSSHPublicKeyOutput
	ListServiceSpecificCredentialsOutput  iam.ListServiceSpecificCredentialsOutput
	DeleteServiceSpecificCredentialOutput iam.DeleteServiceSpecificCredentialOutput
	ListMFADevicesOutput                  iam.ListMFADevicesOutput
	DeactivateMFADeviceOutput             iam.DeactivateMFADeviceOutput
	DeleteVirtualMFADeviceOutput          iam.DeleteVirtualMFADeviceOutput
	DeleteUserOutput                      iam.DeleteUserOutput
}

func (m mockedIAMUsers) ListUsersPagesWithContext(_ aws.Context, input *iam.ListUsersInput, callback func(*iam.ListUsersOutput, bool) bool, _ ...request.Option) error {
	callback(&m.ListUsersPagesOutput, true)
	return nil
}

func (m mockedIAMUsers) ListAttachedUserPoliciesWithContext(_ aws.Context, input *iam.ListAttachedUserPoliciesInput, _ ...request.Option) (*iam.ListAttachedUserPoliciesOutput, error) {
	return &m.ListAttachedUserPoliciesOutput, nil
}

func (m mockedIAMUsers) DetachUserPolicyWithContext(_ aws.Context, input *iam.DetachUserPolicyInput, _ ...request.Option) (*iam.DetachUserPolicyOutput, error) {
	return &m.DetachUserPolicyOutput, nil
}

func (m mockedIAMUsers) ListUserPoliciesWithContext(_ aws.Context, input *iam.ListUserPoliciesInput, _ ...request.Option) (*iam.ListUserPoliciesOutput, error) {
	return &m.ListUserPoliciesOutput, nil
}

func (m mockedIAMUsers) DeleteUserPolicyWithContext(_ aws.Context, input *iam.DeleteUserPolicyInput, _ ...request.Option) (*iam.DeleteUserPolicyOutput, error) {
	return &m.DeleteUserPolicyOutput, nil
}

func (m mockedIAMUsers) ListGroupsForUserWithContext(_ aws.Context, input *iam.ListGroupsForUserInput, _ ...request.Option) (*iam.ListGroupsForUserOutput, error) {
	return &m.ListGroupsForUserOutput, nil
}

func (m mockedIAMUsers) RemoveUserFromGroupWithContext(_ aws.Context, input *iam.RemoveUserFromGroupInput, _ ...request.Option) (*iam.RemoveUserFromGroupOutput, error) {
	return &m.RemoveUserFromGroupOutput, nil
}

func (m mockedIAMUsers) DeleteLoginProfileWithContext(_ aws.Context, input *iam.DeleteLoginProfileInput, _ ...request.Option) (*iam.DeleteLoginProfileOutput, error) {
	return &m.DeleteLoginProfileOutput, nil
}

func (m mockedIAMUsers) ListAccessKeysWithContext(_ aws.Context, input *iam.ListAccessKeysInput, _ ...request.Option) (*iam.ListAccessKeysOutput, error) {
	return &m.ListAccessKeysOutput, nil
}

func (m mockedIAMUsers) DeleteAccessKeyWithContext(_ aws.Context, input *iam.DeleteAccessKeyInput, _ ...request.Option) (*iam.DeleteAccessKeyOutput, error) {
	return &m.DeleteAccessKeyOutput, nil
}

func (m mockedIAMUsers) ListSigningCertificatesWithContext(_ aws.Context, input *iam.ListSigningCertificatesInput, _ ...request.Option) (*iam.ListSigningCertificatesOutput, error) {
	return &m.ListSigningCertificatesOutput, nil
}

func (m mockedIAMUsers) DeleteSigningCertificateWithContext(_ aws.Context, input *iam.DeleteSigningCertificateInput, _ ...request.Option) (*iam.DeleteSigningCertificateOutput, error) {
	return &m.DeleteSigningCertificateOutput, nil
}

func (m mockedIAMUsers) ListSSHPublicKeysWithContext(_ aws.Context, input *iam.ListSSHPublicKeysInput, _ ...request.Option) (*iam.ListSSHPublicKeysOutput, error) {
	return &m.ListSSHPublicKeysOutput, nil
}

func (m mockedIAMUsers) DeleteSSHPublicKeyWithContext(_ aws.Context, input *iam.DeleteSSHPublicKeyInput, _ ...request.Option) (*iam.DeleteSSHPublicKeyOutput, error) {
	return &m.DeleteSSHPublicKeyOutput, nil
}

func (m mockedIAMUsers) ListServiceSpecificCredentialsWithContext(_ aws.Context, input *iam.ListServiceSpecificCredentialsInput, _ ...request.Option) (*iam.ListServiceSpecificCredentialsOutput, error) {
	return &m.ListServiceSpecificCredentialsOutput, nil
}

func (m mockedIAMUsers) DeleteServiceSpecificCredentialWithContext(_ aws.Context, input *iam.DeleteServiceSpecificCredentialInput, _ ...request.Option) (*iam.DeleteServiceSpecificCredentialOutput, error) {
	return &m.DeleteServiceSpecificCredentialOutput, nil
}

func (m mockedIAMUsers) ListMFADevicesWithContext(_ aws.Context, input *iam.ListMFADevicesInput, _ ...request.Option) (*iam.ListMFADevicesOutput, error) {
	return &m.ListMFADevicesOutput, nil
}

func (m mockedIAMUsers) DeactivateMFADeviceWithContext(_ aws.Context, input *iam.DeactivateMFADeviceInput, _ ...request.Option) (*iam.DeactivateMFADeviceOutput, error) {
	return &m.DeactivateMFADeviceOutput, nil
}

func (m mockedIAMUsers) DeleteVirtualMFADeviceWithContext(_ aws.Context, input *iam.DeleteVirtualMFADeviceInput, _ ...request.Option) (*iam.DeleteVirtualMFADeviceOutput, error) {
	return &m.DeleteVirtualMFADeviceOutput, nil
}

func (m mockedIAMUsers) DeleteUserWithContext(_ aws.Context, input *iam.DeleteUserInput, _ ...request.Option) (*iam.DeleteUserOutput, error) {
	return &m.DeleteUserOutput, nil
}

func TestIAMUsers_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	testName1 := "test-user1"
	testName2 := "test-user2"
	iu := IAMUsers{
		Client: mockedIAMUsers{
			ListUsersPagesOutput: iam.ListUsersOutput{
				Users: []*iam.User{
					{
						UserName:   awsgo.String(testName1),
						CreateDate: awsgo.Time(now),
					},
					{
						UserName:   awsgo.String(testName2),
						CreateDate: awsgo.Time(now.Add(1)),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestIAMUsers_NukeAll(t *testing.T) {

	t.Parallel()

	iu := IAMUsers{
		Client: mockedIAMUsers{
			ListAttachedUserPoliciesOutput: iam.ListAttachedUserPoliciesOutput{
				AttachedPolicies: []*iam.AttachedPolicy{
					{
						PolicyName: awsgo.String("test-policy"),
					},
				},
			},
			DetachUserPolicyOutput: iam.DetachUserPolicyOutput{},
			ListUserPoliciesOutput: iam.ListUserPoliciesOutput{
				PolicyNames: []*string{
					awsgo.String("test-policy"),
				},
			},
			DeleteUserPolicyOutput: iam.DeleteUserPolicyOutput{},
			ListGroupsForUserOutput: iam.ListGroupsForUserOutput{
				Groups: []*iam.Group{
					{
						GroupName: awsgo.String("test-group"),
					},
				},
			},
			RemoveUserFromGroupOutput: iam.RemoveUserFromGroupOutput{},
			DeleteLoginProfileOutput:  iam.DeleteLoginProfileOutput{},
			ListAccessKeysOutput: iam.ListAccessKeysOutput{
				AccessKeyMetadata: []*iam.AccessKeyMetadata{
					{
						AccessKeyId: awsgo.String("test-key"),
					},
				},
			},
			DeleteAccessKeyOutput: iam.DeleteAccessKeyOutput{},
			ListSigningCertificatesOutput: iam.ListSigningCertificatesOutput{
				Certificates: []*iam.SigningCertificate{
					{
						CertificateId: awsgo.String("test-certificate"),
					},
				},
			},
			DeleteSigningCertificateOutput: iam.DeleteSigningCertificateOutput{},
			ListSSHPublicKeysOutput: iam.ListSSHPublicKeysOutput{
				SSHPublicKeys: []*iam.SSHPublicKeyMetadata{
					{
						SSHPublicKeyId: awsgo.String("test-ssh-key"),
					},
				},
			},
			DeleteSSHPublicKeyOutput: iam.DeleteSSHPublicKeyOutput{},
			ListServiceSpecificCredentialsOutput: iam.ListServiceSpecificCredentialsOutput{
				ServiceSpecificCredentials: []*iam.ServiceSpecificCredentialMetadata{
					{
						ServiceSpecificCredentialId: awsgo.String("test-service-credential"),
					},
				},
			},
			DeleteServiceSpecificCredentialOutput: iam.DeleteServiceSpecificCredentialOutput{},
			ListMFADevicesOutput: iam.ListMFADevicesOutput{
				MFADevices: []*iam.MFADevice{
					{
						SerialNumber: awsgo.String("test-mfa-device"),
					},
				},
			},
			DeactivateMFADeviceOutput:    iam.DeactivateMFADeviceOutput{},
			DeleteVirtualMFADeviceOutput: iam.DeleteVirtualMFADeviceOutput{},
			DeleteUserOutput:             iam.DeleteUserOutput{},
		},
	}

	err := iu.nukeAll([]*string{awsgo.String("test-user")})
	require.NoError(t, err)
}

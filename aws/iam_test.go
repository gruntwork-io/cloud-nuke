package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
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

func (m mockedIAMUsers) ListUsersPages(input *iam.ListUsersInput, callback func(*iam.ListUsersOutput, bool) bool) error {
	callback(&m.ListUsersPagesOutput, true)
	return nil
}

func (m mockedIAMUsers) ListAttachedUserPolicies(input *iam.ListAttachedUserPoliciesInput) (*iam.ListAttachedUserPoliciesOutput, error) {
	return &m.ListAttachedUserPoliciesOutput, nil
}

func (m mockedIAMUsers) DetachUserPolicy(input *iam.DetachUserPolicyInput) (*iam.DetachUserPolicyOutput, error) {
	return &m.DetachUserPolicyOutput, nil
}

func (m mockedIAMUsers) ListUserPolicies(input *iam.ListUserPoliciesInput) (*iam.ListUserPoliciesOutput, error) {
	return &m.ListUserPoliciesOutput, nil
}

func (m mockedIAMUsers) DeleteUserPolicy(input *iam.DeleteUserPolicyInput) (*iam.DeleteUserPolicyOutput, error) {
	return &m.DeleteUserPolicyOutput, nil
}

func (m mockedIAMUsers) ListGroupsForUser(input *iam.ListGroupsForUserInput) (*iam.ListGroupsForUserOutput, error) {
	return &m.ListGroupsForUserOutput, nil
}

func (m mockedIAMUsers) RemoveUserFromGroup(input *iam.RemoveUserFromGroupInput) (*iam.RemoveUserFromGroupOutput, error) {
	return &m.RemoveUserFromGroupOutput, nil
}

func (m mockedIAMUsers) DeleteLoginProfile(input *iam.DeleteLoginProfileInput) (*iam.DeleteLoginProfileOutput, error) {
	return &m.DeleteLoginProfileOutput, nil
}

func (m mockedIAMUsers) ListAccessKeys(input *iam.ListAccessKeysInput) (*iam.ListAccessKeysOutput, error) {
	return &m.ListAccessKeysOutput, nil
}

func (m mockedIAMUsers) DeleteAccessKey(input *iam.DeleteAccessKeyInput) (*iam.DeleteAccessKeyOutput, error) {
	return &m.DeleteAccessKeyOutput, nil
}

func (m mockedIAMUsers) ListSigningCertificates(input *iam.ListSigningCertificatesInput) (*iam.ListSigningCertificatesOutput, error) {
	return &m.ListSigningCertificatesOutput, nil
}

func (m mockedIAMUsers) DeleteSigningCertificate(input *iam.DeleteSigningCertificateInput) (*iam.DeleteSigningCertificateOutput, error) {
	return &m.DeleteSigningCertificateOutput, nil
}

func (m mockedIAMUsers) ListSSHPublicKeys(input *iam.ListSSHPublicKeysInput) (*iam.ListSSHPublicKeysOutput, error) {
	return &m.ListSSHPublicKeysOutput, nil
}

func (m mockedIAMUsers) DeleteSSHPublicKey(input *iam.DeleteSSHPublicKeyInput) (*iam.DeleteSSHPublicKeyOutput, error) {
	return &m.DeleteSSHPublicKeyOutput, nil
}

func (m mockedIAMUsers) ListServiceSpecificCredentials(input *iam.ListServiceSpecificCredentialsInput) (*iam.ListServiceSpecificCredentialsOutput, error) {
	return &m.ListServiceSpecificCredentialsOutput, nil
}

func (m mockedIAMUsers) DeleteServiceSpecificCredential(input *iam.DeleteServiceSpecificCredentialInput) (*iam.DeleteServiceSpecificCredentialOutput, error) {
	return &m.DeleteServiceSpecificCredentialOutput, nil
}

func (m mockedIAMUsers) ListMFADevices(input *iam.ListMFADevicesInput) (*iam.ListMFADevicesOutput, error) {
	return &m.ListMFADevicesOutput, nil
}

func (m mockedIAMUsers) DeactivateMFADevice(input *iam.DeactivateMFADeviceInput) (*iam.DeactivateMFADeviceOutput, error) {
	return &m.DeactivateMFADeviceOutput, nil
}

func (m mockedIAMUsers) DeleteVirtualMFADevice(input *iam.DeleteVirtualMFADeviceInput) (*iam.DeleteVirtualMFADeviceOutput, error) {
	return &m.DeleteVirtualMFADeviceOutput, nil
}

func (m mockedIAMUsers) DeleteUser(input *iam.DeleteUserInput) (*iam.DeleteUserOutput, error) {
	return &m.DeleteUserOutput, nil
}

func TestIAMUsers_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
			names, err := iu.getAll(config.Config{
				IAMUsers: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestIAMUsers_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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

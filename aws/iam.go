package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAllIAMUsers(session *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := iam.New(session)

	result, err := svc.ListUsers(&iam.ListUsersInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var users []*string

	for _, user := range result.Users {
		if user.CreateDate != nil && excludeAfter.After(awsgo.TimeValue(user.CreateDate)) {
			users = append(users, user.UserName)
		}
	}

	return users, nil
}

func nukeAllIamUsers(session *session.Session, users []*string) error {
	svc := iam.New(session)

	if len(users) == 0 {
		logging.Logger.Infof("No IAM Users to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all IAM Users in region %s", *session.Config.Region)
	usersToDelete := []*string{}

	for _, user := range users {

		// Delete Password
		params := &iam.DeleteLoginProfileInput{
			UserName: user,
		}

		_, err := svc.DeleteLoginProfile(params)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		// Delete Access Keys
		accessKeys, err := svc.ListAccessKeys(&iam.ListAccessKeysInput{
			UserName: user,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		if len(accessKeys.AccessKeyMetadata) > 0 {
			for _, metadata := range accessKeys.AccessKeyMetadata {

				_, err := svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{
					AccessKeyId: metadata.AccessKeyId,
					UserName:    user,
				})

				if err != nil {
					logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(metadata.AccessKeyId), err)
				}
			}
		}

		// Delete Signing Certificates
		certificates, err := svc.ListSigningCertificates(&iam.ListSigningCertificatesInput{
			UserName: user,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		if len(certificates.Certificates) > 0 {
			for _, certificate := range certificates.Certificates {

				_, err := svc.DeleteSigningCertificate(&iam.DeleteSigningCertificateInput{
					CertificateId: certificate.CertificateId,
				})

				if err != nil {
					logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(certificate.CertificateId), err)
				}
			}
		}

		// Delete SSH public Keys
		sshKeys, err := svc.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		if len(sshKeys.SSHPublicKeys) > 0 {
			for _, sshKey := range sshKeys.SSHPublicKeys {
				_, err := svc.DeleteSSHPublicKey(&iam.DeleteSSHPublicKeyInput{
					SSHPublicKeyId: sshKey.SSHPublicKeyId,
					UserName:       user,
				})

				if err != nil {
					logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(sshKey.SSHPublicKeyId), err)
				}
			}
		}

		// Delete Git credentials
		credentials, err := svc.ListServiceSpecificCredentials(&iam.ListServiceSpecificCredentialsInput{
			UserName: user,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		if len(credentials.ServiceSpecificCredentials) > 0 {
			for _, credential := range credentials.ServiceSpecificCredentials {
				_, err := svc.DeleteServiceSpecificCredential(&iam.DeleteServiceSpecificCredentialInput{
					ServiceSpecificCredentialId: credential.ServiceSpecificCredentialId,
				})

				if err != nil {
					logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(credential.ServiceSpecificCredentialId), err)
				}
			}
		}

		// Delete Multi-factor authentication (MFA) device
		mfaDevices, err := svc.ListMFADevices(&iam.ListMFADevicesInput{
			UserName: user,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		if len(mfaDevices.MFADevices) > 0 {
			for _, mfaDevice := range mfaDevices.MFADevices {
				// To delete the MFA device we have to deativate it first
				_, err := svc.DeactivateMFADevice(&iam.DeactivateMFADeviceInput{
					SerialNumber: mfaDevice.SerialNumber,
					UserName:     user,
				})

				if err != nil {
					logging.Logger.Errorf("[Failed] %s: %s", mfaDevice.SerialNumber, err)

				} else {
					_, err := svc.DeleteVirtualMFADevice(&iam.DeleteVirtualMFADeviceInput{
						SerialNumber: mfaDevice.SerialNumber,
					})

					if err != nil {
						logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(mfaDevice.SerialNumber), err)
					}
				}
			}
		}

		// Delete Inline policies
		inlinePolicies, err := svc.ListUserPolicies(&iam.ListUserPoliciesInput{
			UserName: user,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		if len(inlinePolicies.PolicyNames) > 0 {
			for _, inlinePolicy := range inlinePolicies.PolicyNames {
				_, err := svc.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
					PolicyName: inlinePolicy,
					UserName:   user,
				})
				if err != nil {
					logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(inlinePolicy), err)
				}
			}
		}

		// Detach managed policies
		managedPolicies, err := svc.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
			UserName: user,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		if len(managedPolicies.AttachedPolicies) > 0 {
			for _, managedPolicy := range managedPolicies.AttachedPolicies {
				_, err := svc.DetachUserPolicy(&iam.DetachUserPolicyInput{
					PolicyArn: managedPolicy.PolicyArn,
					UserName:  user,
				})

				if err != nil {
					logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(managedPolicy.PolicyArn), err)
				}
			}
		}

		// Remove user from Group Memberships
		userGroups, err := svc.ListGroupsForUser(&iam.ListGroupsForUserInput{
			UserName: user,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		if len(userGroups.Groups) > 0 {
			for _, group := range userGroups.Groups {
				_, err := svc.RemoveUserFromGroup(&iam.RemoveUserFromGroupInput{
					GroupName: group.GroupName,
					UserName:  user,
				})

				if err != nil {
					logging.Logger.Errorf("[Failed] %s: %s", awsgo.StringValue(group.GroupName), err)
				}
			}
		}

	usersToDelete = append(usersToDelete, user)
	}

	deletedUsers := []*string{}
	if len(usersToDelete) > 0 {
		for _, user := range usersToDelete {

			_, err := svc.DeleteUser(&iam.DeleteUserInput{
				UserName: user,
			})

			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
			deletedUsers = append(deletedUsers, user)
		}
	}

	if len(deletedUsers) != len(users) {
		logging.Logger.Errorf("[Failed] - %d/%d - IAM User(s) failed deletion in %s", len(users)-len(deletedUsers), len(users), *session.Config.Region)
	}

	logging.Logger.Infof("[OK] %d IAM User(s) deleted in %s", len(deletedUsers), *session.Config.Region)
	return nil
}

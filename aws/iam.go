package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"
	"github.com/hashicorp/go-multierror"
)

// List all IAM users in the AWS account and returns a slice of the UserNames
func getAllIamUsers(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)
	input := &iam.ListUsersInput{}

	var userNames []*string

	// TODO: Probably use ListUsers together with ListUsersPages in case there are lots of users
	output, err := svc.ListUsers(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, user := range output.Users {
		if config.ShouldInclude(aws.StringValue(user.UserName), configObj.IAMUsers.IncludeRule.NamesRegExp, configObj.IAMUsers.ExcludeRule.NamesRegExp) && excludeAfter.After(*user.CreateDate) {
			userNames = append(userNames, user.UserName)
		}
	}

	return userNames, nil
}

func getAllIamRoles(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)

	allIAMRoles := []*string{}
	err := svc.ListRolesPages(
		&iam.ListRolesInput{},
		func(page *iam.ListRolesOutput, lastPage bool) bool {
			for _, iamRole := range page.Roles {
				if shouldIncludeIAMRole(iamRole, excludeAfter, configObj) {
					allIAMRoles = append(allIAMRoles, iamRole.RoleName)
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return allIAMRoles, nil
}

func shouldIncludeIAMRole(iamRole *iam.Role, excludeAfter time.Time, configObj config.Config) bool {
	if iamRole == nil {
		return false
	}

	if excludeAfter.Before(*iamRole.CreateDate) {
		return false
	}

	return true
}

func detachUserPolicies(svc *iam.IAM, userName *string) error {
	policiesOutput, err := svc.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: userName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, attachedPolicy := range policiesOutput.AttachedPolicies {
		arn := attachedPolicy.PolicyArn
		_, err = svc.DetachUserPolicy(&iam.DetachUserPolicyInput{
			PolicyArn: arn,
			UserName:  userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Detached Policy %s from User %s", aws.StringValue(arn), aws.StringValue(userName))
	}

	return nil
}

func detachInstanceProfilesFromRole(svc *iam.IAM, roleName *string) error {
	profilesOutput, err := svc.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, associatedInstanceProfile := range profilesOutput.InstanceProfiles {
		instanceProfileName := associatedInstanceProfile.InstanceProfileName
		_, err := svc.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: instanceProfileName,
			RoleName:            roleName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Detached InstanceProfile %s from Role %s", aws.StringValue(instanceProfileName), aws.StringValue(roleName))
	}
	return nil
}

func deleteInlineUserPolicies(svc *iam.IAM, userName *string) error {
	policyOutput, err := svc.ListUserPolicies(&iam.ListUserPoliciesInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, policyName := range policyOutput.PolicyNames {
		_, err := svc.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
			PolicyName: policyName,
			UserName:   userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Deleted Inline Policy %s from User %s", aws.StringValue(policyName), aws.StringValue(userName))
	}

	return nil
}

func deleteInlineRolePolicies(svc *iam.IAM, roleName *string) error {
	policyOutput, err := svc.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: roleName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, policyName := range policyOutput.PolicyNames {
		_, err := svc.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			PolicyName: policyName,
			RoleName:   roleName,
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				logging.Logger.Infof("Got error code: %s in deleteInlineRolePolicies", awsErr.Code())
				if awsErr.Code() == iam.ErrCodeUnmodifiableEntityException {
					logging.Logger.Infof("Skipping removal of inline policies from IAM role %s because it is an AWS-Managed role, which cannot be modified by users", aws.StringValue(roleName))
					return nil
				}
			}
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Deleted Inline Policy %s from Role %s", aws.StringValue(policyName), aws.StringValue(roleName))
	}

	return nil
}

// deleteManagedRolePolicies will enumerate and delete any *managed* policies attached to the role. Managed IAM policies are those
// maintained by AWS. They must be handled separately from *inline* policies that the user may have attached to the role
func deleteManagedRolePolicies(svc *iam.IAM, roleName *string) error {
	policyOutput, err := svc.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: roleName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, attachedPolicy := range policyOutput.AttachedPolicies {
		_, err := svc.DetachRolePolicy(&iam.DetachRolePolicyInput{
			PolicyArn: attachedPolicy.PolicyArn,
			RoleName:  roleName,
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == iam.ErrCodeUnmodifiableEntityException {
					logging.Logger.Infof("Skipping removal of managed policies from IAM role %s because it is an AWS-Managed role, which cannot be modified by users", aws.StringValue(roleName))
					return nil
				}
			}
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Deleted Inline Policy %s from Role %s", aws.StringValue(attachedPolicy.PolicyName), aws.StringValue(roleName))
	}

	return nil
}

func removeUserFromGroups(svc *iam.IAM, userName *string) error {
	groupsOutput, err := svc.ListGroupsForUser(&iam.ListGroupsForUserInput{
		UserName: userName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, group := range groupsOutput.Groups {
		_, err := svc.RemoveUserFromGroup(&iam.RemoveUserFromGroupInput{
			GroupName: group.GroupName,
			UserName:  userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Removed user %s from group %s", aws.StringValue(userName), aws.StringValue(group.GroupName))
	}

	return nil
}

func deleteLoginProfile(svc *iam.IAM, userName *string) error {
	return retry.DoWithRetry(
		logging.Logger,
		"Delete Login Profile",
		10,
		2*time.Second,
		func() error {
			// Delete Login Profile attached to the user
			_, err := svc.DeleteLoginProfile(&iam.DeleteLoginProfileInput{
				UserName: userName,
			})
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case iam.ErrCodeNoSuchEntityException:
						// This is expected if the user doesn't have a Login Profile
						// (automated users created via API calls withouth further
						// configuration)
						return nil
					case iam.ErrCodeEntityTemporarilyUnmodifiableException:
						// The request was rejected because it referenced an entity that is
						// temporarily unmodifiable. We have to try again.
						return fmt.Errorf("Login Profile for user %s cannot be deleted now", aws.StringValue(userName))
					default:
						return retry.FatalError{Underlying: err}
					}
				}
			}

			logging.Logger.Infof("Deleted Login Profile from user %s", aws.StringValue(userName))
			return nil
		})
}

func deleteAccessKeys(svc *iam.IAM, userName *string) error {
	output, err := svc.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, md := range output.AccessKeyMetadata {
		accessKeyId := md.AccessKeyId
		_, err := svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			AccessKeyId: accessKeyId,
			UserName:    userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Deleted Access Key %s from user %s", aws.StringValue(accessKeyId), aws.StringValue(userName))
	}

	return nil
}

func deleteSigningCertificate(svc *iam.IAM, userName *string) error {
	output, err := svc.ListSigningCertificates(&iam.ListSigningCertificatesInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, cert := range output.Certificates {
		certificateId := cert.CertificateId
		_, err := svc.DeleteSigningCertificate(&iam.DeleteSigningCertificateInput{
			CertificateId: certificateId,
			UserName:      userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Deleted Signing Certificate ID %s from user %s", aws.StringValue(certificateId), aws.StringValue(userName))
	}

	return nil
}

func deleteSSHPublicKeys(svc *iam.IAM, userName *string) error {
	output, err := svc.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, key := range output.SSHPublicKeys {
		keyId := key.SSHPublicKeyId
		_, err := svc.DeleteSSHPublicKey(&iam.DeleteSSHPublicKeyInput{
			SSHPublicKeyId: keyId,
			UserName:       userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Deleted SSH Public Key with ID %s from user %s", aws.StringValue(keyId), aws.StringValue(userName))
	}

	return nil
}

func deleteServiceSpecificCredentials(svc *iam.IAM, userName *string) error {
	services := []string{
		"cassandra.amazonaws.com",
		"codecommit.amazonaws.com",
	}
	for _, service := range services {
		output, err := svc.ListServiceSpecificCredentials(&iam.ListServiceSpecificCredentialsInput{
			ServiceName: aws.String(service),
			UserName:    userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		for _, metadata := range output.ServiceSpecificCredentials {
			serviceSpecificCredentialId := metadata.ServiceSpecificCredentialId

			_, err := svc.DeleteServiceSpecificCredential(&iam.DeleteServiceSpecificCredentialInput{
				ServiceSpecificCredentialId: serviceSpecificCredentialId,
				UserName:                    userName,
			})
			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}

			logging.Logger.Infof("Deleted Service Specific Credential with ID %s of service %s from user %s", aws.StringValue(serviceSpecificCredentialId), service, aws.StringValue(userName))
		}
	}

	return nil
}

func deleteMFADevices(svc *iam.IAM, userName *string) error {
	output, err := svc.ListMFADevices(&iam.ListMFADevicesInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	// First we need to deactivate the devices
	for _, device := range output.MFADevices {
		serialNumber := device.SerialNumber

		_, err := svc.DeactivateMFADevice(&iam.DeactivateMFADeviceInput{
			SerialNumber: serialNumber,
			UserName:     userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Deactivated Virtual MFA Device with ID %s from user %s", aws.StringValue(serialNumber), aws.StringValue(userName))
	}

	// After their deactivation we can delete them
	for _, device := range output.MFADevices {
		serialNumber := device.SerialNumber

		_, err := svc.DeleteVirtualMFADevice(&iam.DeleteVirtualMFADeviceInput{
			SerialNumber: serialNumber,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Deleted Virtual MFA Device with ID %s from user %s", aws.StringValue(serialNumber), aws.StringValue(userName))
	}

	return nil
}

func deleteUser(svc *iam.IAM, userName *string) error {
	_, err := svc.DeleteUser(&iam.DeleteUserInput{
		UserName: userName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func deleteIamRole(svc *iam.IAM, roleName *string) error {
	_, err := svc.DeleteRole(&iam.DeleteRoleInput{
		RoleName: roleName,
	})
	if err != nil {

		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == iam.ErrCodeUnmodifiableEntityException {
				logging.Logger.Infof("Skipping deletion of IAM role %s because it is an AWS-Managed role, which cannot be modified by users", aws.StringValue(roleName))
				return ManagedIAMRoleErr{Name: aws.StringValue(roleName)}
			}
		}

		return errors.WithStackTrace(err)
	}

	return nil
}

// Nuke a single user
func nukeUser(svc *iam.IAM, userName *string) error {
	// Functions used to really nuke an IAM User as a user can have many attached
	// items we need delete/detach them before actually deleting it.
	// NOTE: The actual user deletion should always be the last one. This way we
	// can guarantee that it will fail if we forgot to delete/detach an item.
	functions := []func(svc *iam.IAM, userName *string) error{
		detachUserPolicies, // TODO: Add CLI option to delete the Policy as policies exist independently of the user
		deleteInlineUserPolicies,
		removeUserFromGroups, // TODO: Add CLI option to delete groups as groups exist independently of the user
		deleteLoginProfile,
		deleteAccessKeys,
		deleteSigningCertificate,
		deleteSSHPublicKeys,
		deleteServiceSpecificCredentials,
		deleteMFADevices,
		deleteUser,
	}

	for _, fn := range functions {
		if err := fn(svc, userName); err != nil {
			return err
		}
	}

	return nil
}

// Delete all IAM Users
func nukeAllIamUsers(session *session.Session, userNames []*string) error {
	if len(userNames) == 0 {
		logging.Logger.Info("No IAM Users to nuke")

		return nil
	}

	logging.Logger.Info("Deleting all IAM Users")

	deletedUsers := 0
	svc := iam.New(session)
	multiErr := new(multierror.Error)

	for _, userName := range userNames {
		err := nukeUser(svc, userName)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		} else {
			deletedUsers++
			logging.Logger.Infof("Deleted IAM User: %s", *userName)
		}
	}

	logging.Logger.Infof("[OK] %d IAM User(s) terminated", deletedUsers)
	return multiErr.ErrorOrNil()
}

// Nuke a single role
func nukeRole(svc *iam.IAM, roleName *string) error {
	// Functions used to really nuke an IAM Role as a role can have many attached
	// items we need delete/detach them before actually deleting it.
	// NOTE: The actual role deletion should always be the last one. This way we
	// can guarantee that it will fail if we forgot to delete/detach an item.
	functions := []func(svc *iam.IAM, userName *string) error{
		detachInstanceProfilesFromRole,
		deleteInlineRolePolicies,
		deleteManagedRolePolicies,
		deleteIamRole,
	}

	for _, fn := range functions {
		if err := fn(svc, roleName); err != nil {
			return err
		}
	}

	return nil
} // Delete all IAM Roles
func nukeAllIamRoles(session *session.Session, roleNames []*string) error {
	if len(roleNames) == 0 {
		logging.Logger.Info("No IAM Roles to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all IAM Roles")

	deletedRoles := 0
	svc := iam.New(session)
	multiErr := new(multierror.Error)

	for _, roleName := range roleNames {
		switch err := nukeRole(svc, roleName); err.(type) {
		case nil:
			deletedRoles++
			logging.Logger.Infof("Deleted IAM Role: %s", aws.StringValue(roleName))
		case *ManagedIAMRoleErr:
			logging.Logger.Infof("Skipped IAM Role: %s because it is an AWS-managed role. Only AWS can modify / delete AWS-managed roles", aws.StringValue(roleName))
		default:
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		}
	}

	logging.Logger.Infof("[OK] %d IAM Role(s) terminated", deletedRoles)
	return multiErr.ErrorOrNil()
}

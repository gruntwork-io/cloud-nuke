package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"
	"github.com/hashicorp/go-multierror"
)

// List all IAM users in the AWS account and returns a slice of the UserNames
func getAllIamUsers(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)
	input := &iam.ListUsersInput{}

	var userNames []*string

	err := svc.ListUsersPages(input, func(page *iam.ListUsersOutput, lastPage bool) bool {
		for _, user := range page.Users {
			if config.ShouldInclude(aws.StringValue(user.UserName), configObj.IAMUser.IncludeRule.NamesRegExp, configObj.IAMUser.ExcludeRule.NamesRegExp) && excludeAfter.After(*user.CreateDate) {
				userNames = append(userNames, user.UserName)
			}
		}
		return !lastPage
	})
	return userNames, errors.WithStackTrace(err)
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
		logging.Logger.Debugf("Detached Policy %s from User %s", aws.StringValue(arn), aws.StringValue(userName))
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
		logging.Logger.Debugf("Deleted Inline Policy %s from User %s", aws.StringValue(policyName), aws.StringValue(userName))
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
		logging.Logger.Debugf("Removed user %s from group %s", aws.StringValue(userName), aws.StringValue(group.GroupName))
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

			logging.Logger.Debugf("Deleted Login Profile from user %s", aws.StringValue(userName))
			return nil
		})
}

func deleteAccessKeys(svc *iam.IAM, userName *string) error {
	output, err := svc.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, md := range output.AccessKeyMetadata {
		accessKeyId := md.AccessKeyId
		_, err := svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			AccessKeyId: accessKeyId,
			UserName:    userName,
		})
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Debugf("Deleted Access Key %s from user %s", aws.StringValue(accessKeyId), aws.StringValue(userName))
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

		logging.Logger.Debugf("Deleted Signing Certificate ID %s from user %s", aws.StringValue(certificateId), aws.StringValue(userName))
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

		logging.Logger.Debugf("Deleted SSH Public Key with ID %s from user %s", aws.StringValue(keyId), aws.StringValue(userName))
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

			logging.Logger.Debugf("Deleted Service Specific Credential with ID %s of service %s from user %s", aws.StringValue(serviceSpecificCredentialId), service, aws.StringValue(userName))
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

		logging.Logger.Debugf("Deactivated Virtual MFA Device with ID %s from user %s", aws.StringValue(serialNumber), aws.StringValue(userName))
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

		logging.Logger.Debugf("Deleted Virtual MFA Device with ID %s from user %s", aws.StringValue(serialNumber), aws.StringValue(userName))
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
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(userName),
			ResourceType: "IAM User",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking IAM User",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedUsers++
			logging.Logger.Debugf("Deleted IAM User: %s", *userName)
		}
	}

	logging.Logger.Debugf("[OK] %d IAM User(s) terminated", deletedUsers)
	return multiErr.ErrorOrNil()
}

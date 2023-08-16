package resources

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"
	"github.com/hashicorp/go-multierror"
)

// List all IAM users in the AWS account and returns a slice of the UserNames
func (iu *IAMUsers) getAll(configObj config.Config) ([]*string, error) {
	input := &iam.ListUsersInput{}

	var userNames []*string
	err := iu.Client.ListUsersPages(input, func(page *iam.ListUsersOutput, lastPage bool) bool {
		for _, user := range page.Users {
			if configObj.IAMUsers.ShouldInclude(config.ResourceValue{
				Name: user.UserName,
				Time: user.CreateDate,
			}) {
				userNames = append(userNames, user.UserName)
			}
		}

		return !lastPage
	})

	return userNames, errors.WithStackTrace(err)
}

func (iu *IAMUsers) detachUserPolicies(userName *string) error {
	policiesOutput, err := iu.Client.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: userName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, attachedPolicy := range policiesOutput.AttachedPolicies {
		arn := attachedPolicy.PolicyArn
		_, err = iu.Client.DetachUserPolicy(&iam.DetachUserPolicyInput{
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

func (iu *IAMUsers) deleteInlineUserPolicies(userName *string) error {
	policyOutput, err := iu.Client.ListUserPolicies(&iam.ListUserPoliciesInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, policyName := range policyOutput.PolicyNames {
		_, err := iu.Client.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
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

func (iu *IAMUsers) removeUserFromGroups(userName *string) error {
	groupsOutput, err := iu.Client.ListGroupsForUser(&iam.ListGroupsForUserInput{
		UserName: userName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, group := range groupsOutput.Groups {
		_, err := iu.Client.RemoveUserFromGroup(&iam.RemoveUserFromGroupInput{
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

func (iu *IAMUsers) deleteLoginProfile(userName *string) error {
	return retry.DoWithRetry(
		logging.Logger,
		"Delete Login Profile",
		10,
		2*time.Second,
		func() error {
			// Delete Login Profile attached to the user
			_, err := iu.Client.DeleteLoginProfile(&iam.DeleteLoginProfileInput{
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

func (iu *IAMUsers) deleteAccessKeys(userName *string) error {
	output, err := iu.Client.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, md := range output.AccessKeyMetadata {
		accessKeyId := md.AccessKeyId
		_, err := iu.Client.DeleteAccessKey(&iam.DeleteAccessKeyInput{
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

func (iu *IAMUsers) deleteSigningCertificate(userName *string) error {
	output, err := iu.Client.ListSigningCertificates(&iam.ListSigningCertificatesInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, cert := range output.Certificates {
		certificateId := cert.CertificateId
		_, err := iu.Client.DeleteSigningCertificate(&iam.DeleteSigningCertificateInput{
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

func (iu *IAMUsers) deleteSSHPublicKeys(userName *string) error {
	output, err := iu.Client.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, key := range output.SSHPublicKeys {
		keyId := key.SSHPublicKeyId
		_, err := iu.Client.DeleteSSHPublicKey(&iam.DeleteSSHPublicKeyInput{
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

func (iu *IAMUsers) deleteServiceSpecificCredentials(userName *string) error {
	services := []string{
		"cassandra.amazonaws.com",
		"codecommit.amazonaws.com",
	}
	for _, service := range services {
		output, err := iu.Client.ListServiceSpecificCredentials(&iam.ListServiceSpecificCredentialsInput{
			ServiceName: aws.String(service),
			UserName:    userName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		for _, metadata := range output.ServiceSpecificCredentials {
			serviceSpecificCredentialId := metadata.ServiceSpecificCredentialId

			_, err := iu.Client.DeleteServiceSpecificCredential(&iam.DeleteServiceSpecificCredentialInput{
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

func (iu *IAMUsers) deleteMFADevices(userName *string) error {
	output, err := iu.Client.ListMFADevices(&iam.ListMFADevicesInput{
		UserName: userName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	// First we need to deactivate the devices
	for _, device := range output.MFADevices {
		serialNumber := device.SerialNumber

		_, err := iu.Client.DeactivateMFADevice(&iam.DeactivateMFADeviceInput{
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

		_, err := iu.Client.DeleteVirtualMFADevice(&iam.DeleteVirtualMFADeviceInput{
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

func (iu *IAMUsers) deleteUser(userName *string) error {
	_, err := iu.Client.DeleteUser(&iam.DeleteUserInput{
		UserName: userName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Nuke a single user
func (iu *IAMUsers) nukeUser(userName *string) error {
	// Functions used to really nuke an IAM User as a user can have many attached
	// items we need delete/detach them before actually deleting it.
	// NOTE: The actual user deletion should always be the last one. This way we
	// can guarantee that it will fail if we forgot to delete/detach an item.
	functions := []func(userName *string) error{
		iu.detachUserPolicies, // TODO: Add CLI option to delete the Policy as policies exist independently of the user
		iu.deleteInlineUserPolicies,
		iu.removeUserFromGroups, // TODO: Add CLI option to delete groups as groups exist independently of the user
		iu.deleteLoginProfile,
		iu.deleteAccessKeys,
		iu.deleteSigningCertificate,
		iu.deleteSSHPublicKeys,
		iu.deleteServiceSpecificCredentials,
		iu.deleteMFADevices,
		iu.deleteUser,
	}

	for _, fn := range functions {
		if err := fn(userName); err != nil {
			return err
		}
	}

	return nil
}

// Delete all IAM Users
func (iu *IAMUsers) nukeAll(userNames []*string) error {
	if len(userNames) == 0 {
		logging.Logger.Info("No IAM Users to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all IAM Users")

	deletedUsers := 0
	multiErr := new(multierror.Error)

	for _, userName := range userNames {
		err := iu.nukeUser(userName)
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
			}, map[string]interface{}{})
		} else {
			deletedUsers++
			logging.Logger.Debugf("Deleted IAM User: %s", *userName)
		}
	}

	logging.Logger.Debugf("[OK] %d IAM User(s) terminated", deletedUsers)
	return multiErr.ErrorOrNil()
}

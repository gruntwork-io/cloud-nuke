package resources

import (
	"context"
	goerr "errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"
)

// IAMUsersAPI defines the interface for IAM user operations.
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

// NewIAMUsers creates a new IAMUsers resource using the generic resource pattern.
func NewIAMUsers() AwsResource {
	return NewAwsResource(&resource.Resource[IAMUsersAPI]{
		ResourceTypeName: "iam",
		BatchSize:        DefaultBatchSize,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[IAMUsersAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = iam.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.IAMUsers
		},
		Lister: listIAMUsers,
		// IAM deletion requires sequential steps per user - use SequentialDeleter
		Nuker: resource.SequentialDeleter(deleteIAMUser),
	})
}

// listIAMUsers retrieves all IAM users that match the config filters.
func listIAMUsers(ctx context.Context, client IAMUsersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var userNames []*string

	input := &iam.ListUsersInput{}
	paginator := iam.NewListUsersPaginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, user := range page.Users {
			// Note:
			// IAM resource-listing operations return a subset of the available attributes for the resource.
			// This operation does not return the following attributes, even though they are an attribute of the returned object:
			//    PermissionsBoundary
			//    Tags
			// Reference: https://docs.aws.amazon.com/cli/latest/reference/iam/list-users.html

			var tags []types.Tag

			tagsPaginator := iam.NewListUserTagsPaginator(client, &iam.ListUserTagsInput{
				UserName: user.UserName,
			})
			for tagsPaginator.HasMorePages() {
				tagsPage, errListTags := tagsPaginator.NextPage(ctx)
				if errListTags != nil {
					return nil, errors.WithStackTrace(errListTags)
				}

				tags = append(tags, tagsPage.Tags...)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: user.UserName,
				Time: user.CreateDate,
				Tags: util.ConvertIAMTagsToMap(tags),
			}) {
				userNames = append(userNames, user.UserName)
			}
		}
	}

	return userNames, nil
}

// deleteIAMUser deletes an IAM user and all associated resources.
// This function performs all cleanup steps in order:
// 1. Detach user policies
// 2. Delete inline user policies
// 3. Remove user from groups
// 4. Delete login profile
// 5. Delete access keys
// 6. Delete signing certificates
// 7. Delete SSH public keys
// 8. Delete service-specific credentials
// 9. Delete MFA devices
// 10. Delete the user
func deleteIAMUser(ctx context.Context, client IAMUsersAPI, userName *string) error {
	// Step 1: Detach user policies
	if err := detachUserPolicies(ctx, client, userName); err != nil {
		return err
	}

	// Step 2: Delete inline user policies
	if err := deleteInlineUserPolicies(ctx, client, userName); err != nil {
		return err
	}

	// Step 3: Remove user from groups
	if err := removeUserFromGroups(ctx, client, userName); err != nil {
		return err
	}

	// Step 4: Delete login profile
	if err := deleteLoginProfile(ctx, client, userName); err != nil {
		return err
	}

	// Step 5: Delete access keys
	if err := deleteAccessKeys(ctx, client, userName); err != nil {
		return err
	}

	// Step 6: Delete signing certificates
	if err := deleteSigningCertificates(ctx, client, userName); err != nil {
		return err
	}

	// Step 7: Delete SSH public keys
	if err := deleteSSHPublicKeys(ctx, client, userName); err != nil {
		return err
	}

	// Step 8: Delete service-specific credentials
	if err := deleteServiceSpecificCredentials(ctx, client, userName); err != nil {
		return err
	}

	// Step 9: Delete MFA devices
	if err := deleteMFADevices(ctx, client, userName); err != nil {
		return err
	}

	// Step 10: Delete the user
	_, err := client.DeleteUser(ctx, &iam.DeleteUserInput{
		UserName: userName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// detachUserPolicies detaches all managed policies from a user.
func detachUserPolicies(ctx context.Context, client IAMUsersAPI, userName *string) error {
	paginator := iam.NewListAttachedUserPoliciesPaginator(client, &iam.ListAttachedUserPoliciesInput{
		UserName: userName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, attachedPolicy := range page.AttachedPolicies {
			_, err = client.DetachUserPolicy(ctx, &iam.DetachUserPolicyInput{
				PolicyArn: attachedPolicy.PolicyArn,
				UserName:  userName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Detached Policy %s from User %s", aws.ToString(attachedPolicy.PolicyArn), aws.ToString(userName))
		}
	}

	return nil
}

// deleteInlineUserPolicies deletes all inline policies from a user.
func deleteInlineUserPolicies(ctx context.Context, client IAMUsersAPI, userName *string) error {
	paginator := iam.NewListUserPoliciesPaginator(client, &iam.ListUserPoliciesInput{
		UserName: userName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, policyName := range page.PolicyNames {
			_, err := client.DeleteUserPolicy(ctx, &iam.DeleteUserPolicyInput{
				PolicyName: aws.String(policyName),
				UserName:   userName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Deleted Inline Policy %s from User %s", policyName, aws.ToString(userName))
		}
	}

	return nil
}

// removeUserFromGroups removes a user from all groups.
func removeUserFromGroups(ctx context.Context, client IAMUsersAPI, userName *string) error {
	paginator := iam.NewListGroupsForUserPaginator(client, &iam.ListGroupsForUserInput{
		UserName: userName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, group := range page.Groups {
			_, err := client.RemoveUserFromGroup(ctx, &iam.RemoveUserFromGroupInput{
				GroupName: group.GroupName,
				UserName:  userName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Removed user %s from group %s", aws.ToString(userName), aws.ToString(group.GroupName))
		}
	}

	return nil
}

// deleteLoginProfile deletes the login profile for a user.
func deleteLoginProfile(ctx context.Context, client IAMUsersAPI, userName *string) error {
	return retry.DoWithRetry(
		logging.Logger.WithTime(time.Now()),
		"Delete Login Profile",
		10,
		2*time.Second,
		func() error {
			// Delete Login Profile attached to the user
			_, err := client.DeleteLoginProfile(ctx, &iam.DeleteLoginProfileInput{
				UserName: userName,
			})
			if err != nil {
				var (
					errNoSuchEntityException                  *types.NoSuchEntityException
					errEntityTemporarilyUnmodifiableException *types.EntityTemporarilyUnmodifiableException
				)
				switch {
				case goerr.As(err, &errNoSuchEntityException):
					// This is expected if the user doesn't have a Login Profile
					// (automated users created via API calls without further
					// configuration)
					return nil
				case goerr.As(err, &errEntityTemporarilyUnmodifiableException):
					// The request was rejected because it referenced an entity that is
					// temporarily unmodifiable. We have to try again.
					return fmt.Errorf("Login Profile for user %s cannot be deleted now", aws.ToString(userName))
				default:
					return retry.FatalError{Underlying: err}
				}
			}

			logging.Debugf("Deleted Login Profile from user %s", aws.ToString(userName))
			return nil
		})
}

// deleteAccessKeys deletes all access keys for a user.
func deleteAccessKeys(ctx context.Context, client IAMUsersAPI, userName *string) error {
	paginator := iam.NewListAccessKeysPaginator(client, &iam.ListAccessKeysInput{
		UserName: userName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, md := range page.AccessKeyMetadata {
			_, err := client.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
				AccessKeyId: md.AccessKeyId,
				UserName:    userName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Deleted Access Key %s from user %s", aws.ToString(md.AccessKeyId), aws.ToString(userName))
		}
	}

	return nil
}

// deleteSigningCertificates deletes all signing certificates for a user.
func deleteSigningCertificates(ctx context.Context, client IAMUsersAPI, userName *string) error {
	paginator := iam.NewListSigningCertificatesPaginator(client, &iam.ListSigningCertificatesInput{
		UserName: userName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, cert := range page.Certificates {
			_, err := client.DeleteSigningCertificate(ctx, &iam.DeleteSigningCertificateInput{
				CertificateId: cert.CertificateId,
				UserName:      userName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Deleted Signing Certificate ID %s from user %s", aws.ToString(cert.CertificateId), aws.ToString(userName))
		}
	}

	return nil
}

// deleteSSHPublicKeys deletes all SSH public keys for a user.
func deleteSSHPublicKeys(ctx context.Context, client IAMUsersAPI, userName *string) error {
	paginator := iam.NewListSSHPublicKeysPaginator(client, &iam.ListSSHPublicKeysInput{
		UserName: userName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, key := range page.SSHPublicKeys {
			_, err := client.DeleteSSHPublicKey(ctx, &iam.DeleteSSHPublicKeyInput{
				SSHPublicKeyId: key.SSHPublicKeyId,
				UserName:       userName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Deleted SSH Public Key with ID %s from user %s", aws.ToString(key.SSHPublicKeyId), aws.ToString(userName))
		}
	}

	return nil
}

// deleteServiceSpecificCredentials deletes all service-specific credentials for a user.
func deleteServiceSpecificCredentials(ctx context.Context, client IAMUsersAPI, userName *string) error {
	services := []string{
		"cassandra.amazonaws.com",
		"codecommit.amazonaws.com",
	}
	for _, service := range services {
		output, err := client.ListServiceSpecificCredentials(ctx, &iam.ListServiceSpecificCredentialsInput{
			ServiceName: aws.String(service),
			UserName:    userName,
		})
		if err != nil {
			logging.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		for _, metadata := range output.ServiceSpecificCredentials {
			serviceSpecificCredentialId := metadata.ServiceSpecificCredentialId

			_, err := client.DeleteServiceSpecificCredential(ctx, &iam.DeleteServiceSpecificCredentialInput{
				ServiceSpecificCredentialId: serviceSpecificCredentialId,
				UserName:                    userName,
			})
			if err != nil {
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}

			logging.Debugf("Deleted Service Specific Credential with ID %s of service %s from user %s", aws.ToString(serviceSpecificCredentialId), service, aws.ToString(userName))
		}
	}

	return nil
}

// deleteMFADevices deactivates and deletes all MFA devices for a user.
func deleteMFADevices(ctx context.Context, client IAMUsersAPI, userName *string) error {
	// Collect all MFA devices first (need to deactivate before delete)
	var devices []types.MFADevice
	paginator := iam.NewListMFADevicesPaginator(client, &iam.ListMFADevicesInput{
		UserName: userName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}
		devices = append(devices, page.MFADevices...)
	}

	// Deactivate all devices first
	for _, device := range devices {
		_, err := client.DeactivateMFADevice(ctx, &iam.DeactivateMFADeviceInput{
			SerialNumber: device.SerialNumber,
			UserName:     userName,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Deactivated MFA Device %s from user %s", aws.ToString(device.SerialNumber), aws.ToString(userName))
	}

	// Then delete them
	for _, device := range devices {
		_, err := client.DeleteVirtualMFADevice(ctx, &iam.DeleteVirtualMFADeviceInput{
			SerialNumber: device.SerialNumber,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Deleted MFA Device %s from user %s", aws.ToString(device.SerialNumber), aws.ToString(userName))
	}

	return nil
}

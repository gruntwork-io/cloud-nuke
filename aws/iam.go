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
    deletedUsers := []*string{}

    for _, user := range users {

        // Delete User password
        params := &iam.DeleteLoginProfileInput{
            UserName: user,
        }

        _, err := svc.DeleteLoginProfile(params)

        if err != nil {
            logging.Logger.Errorf("[Failed] %s: %s", user, err)
        }

        // Delete IAM User access Key pairs
        keys_output, err := svc.ListAccessKeys(&iam.ListAccessKeysInput{
            UserName: user,
        })

        if err != nil {
            logging.Logger.Errorf("[Failed] %s: %s", keys_output, err)
        }
        
        if len(keys_output.AccessKeyMetadata) > 0 {
            for _, metadata := range keys_output.AccessKeyMetadata {
                
                _, err := svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{
                    AccessKeyId: metadata.AccessKeyId,
                })
    
                if err != nil {
                    logging.Logger.Errorf("[Failed] %s: %s", metadata, err)
                }
            }
        }

        // Delete IAM User Signing Certificates
        certificates_output, err := svc.ListSigningCertificates(&iam.ListSigningCertificatesInput{
            UserName: user,
        })

        if err != nil {
            logging.Logger.Errorf("[Failed] %s: %s", certificates_output, err)
        }
        
        if len(certificates_output.Certificates) > 0 {
            for _, certificate := range certificates_output.Certificates {
                
                _, err := svc.DeleteSigningCertificate(&iam.DeleteSigningCertificateInput{
                    CertificateId: certificate.CertificateId,
                })

                if err != nil {
                    logging.Logger.Errorf("[Failed] %s: %s", certificate, err)
                }
            }
        }

        // Delete SSH public Keys
		sshKeys_output, err := svc.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{})
		
		if err != nil {
            logging.Logger.Errorf("[Failed] %s: %s", certificates_output, err)
        }
        
        if len(sshKeys_output.SSHPublicKeys) > 0 {
            for _, sshKey := range sshKeys_output.SSHPublicKeys {
                _, err := svc.DeleteSSHPublicKey(&iam.DeleteSSHPublicKeyInput{
                    SSHPublicKeyId: sshKey.SSHPublicKeyId,
                    UserName: user,
                })

                if err != nil {
                    logging.Logger.Errorf("[Failed] %s: %s", sshKey, err)
                }
            }
        }

        // Delete service-specific credential
		credentials_output, err := svc.ListServiceSpecificCredentials(&iam.ListServiceSpecificCredentialsInput{})
		if err != nil {
            logging.Logger.Errorf("[Failed] %s: %s", certificates_output, err)
        }
        
        if len(credentials_output.ServiceSpecificCredentials) > 0 {
            for _ , credential := range credentials_output.ServiceSpecificCredentials {
                _, err := svc.DeleteServiceSpecificCredential(&iam.DeleteServiceSpecificCredentialInput{
                    ServiceSpecificCredentialId: credential.ServiceSpecificCredentialId,
                })

                if err != nil {
                    logging.Logger.Errorf("[Failed] %s: %s", credential, err)
                }
            }
        }

        // Delete MFA Device 
        mfaDevices_output, err := svc.ListMFADevices(&iam.ListMFADevicesInput{
            UserName: user,
		})
		
		if err != nil {
            logging.Logger.Errorf("[Failed] %s: %s", certificates_output, err)
        }
        
        if len(mfaDevices_output.MFADevices) > 0 {
            for _, mfaDevice := range mfaDevices_output.MFADevices {
                // Before we delete the MFA device we have to deativate it first
                _, err := svc.DeactivateMFADevice(&iam.DeactivateMFADeviceInput{
                       SerialNumber: mfaDevice.SerialNumber,
                       UserName: user,
                })

                if err != nil {
                    logging.Logger.Errorf("[Failed] %s: %s", mfaDevice, err)
                } else {
                    _, err := svc.DeleteVirtualMFADevice(&iam.DeleteVirtualMFADeviceInput{
                        SerialNumber: mfaDevice.SerialNumber,
                    })

                    if err != nil {
                        logging.Logger.Errorf("[Failed] %s: %s", mfaDevice, err)
                    }
                }
            }
        }
    }
}

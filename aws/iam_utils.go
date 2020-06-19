package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/logging"
)

func createSessionIfNotExists(awsSession *session.Session) *session.Session {
	if awsSession == nil {
		return session.New()
	}
	return awsSession
}

// GetCallerIdentityArn - gets IAM caller identity and returns the ARN
func GetCallerIdentityArn(awsSession *session.Session) (string, error) {
	awsSession = createSessionIfNotExists(awsSession)
	svc := sts.New(awsSession)
	input := &sts.GetCallerIdentityInput{}
	result, err := svc.GetCallerIdentity(input)
	if err != nil {
		return "", err
	}
	return *result.Arn, nil
}

// GetIAMRoleArn returns IAM role arn given a role name
func GetIAMRoleArn(awsSession *session.Session, roleName string) (string, error) {
	awsSession = createSessionIfNotExists(awsSession)
	svc := iam.New(awsSession)

	input := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	result, err := svc.GetRole(input)
	if err != nil {
		return "", err
	}

	return *result.Role.Arn, nil
}

// IAMRoleExists checks if IAM role exists or not
func IAMRoleExists(awsSession *session.Session, roleName string) (bool, error) {
	_, err := GetIAMRoleArn(awsSession, roleName)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return false, nil
			default:
				return false, err
			}
		} else {
			return false, err
		}
	}
	return true, err
}

// CreateIAMRole creates a AWS IAM role if it does not exist
func CreateIAMRole(awsSession *session.Session, roleName string, roleDescription string, assumeRolePolicyDocument string) (string, error) {
	awsSession = createSessionIfNotExists(awsSession)
	svc := iam.New(awsSession)

	roleExists, err := IAMRoleExists(awsSession, roleName)
	if err != nil {
		logging.Logger.Errorf("Failed to check if role - %s - exists - %s", roleName, err.Error())
		return "", err
	}
	if roleExists {
		return GetIAMRoleArn(awsSession, roleName)
	}

	params := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(assumeRolePolicyDocument),
		Description:              aws.String(roleDescription),
		RoleName:                 aws.String(roleName),
	}

	result, err := svc.CreateRole(params)
	if err != nil {
		return "", err
	}

	err = svc.WaitUntilRoleExists(
		&iam.GetRoleInput{
			RoleName: aws.String(roleName),
		},
	)
	if err != nil {
		return "", err
	}

	return *result.Role.Arn, nil
}

// DeleteIAMRole deletes an AWS IAM role if it exists
func DeleteIAMRole(awsSession *session.Session, roleName string) error {
	awsSession = createSessionIfNotExists(awsSession)
	svc := iam.New(awsSession)

	roleExists, err := IAMRoleExists(awsSession, roleName)
	if err != nil {
		logging.Logger.Errorf("Failed to check if role - %s - exists - %s", roleName, err.Error())
		return err
	}
	if !roleExists {
		return nil
	}

	input := &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	}
	_, err = svc.DeleteRole(input)
	return err
}

// CreateIAMRolePolicy creates an inline IAM role policy if it does not exist
func CreateIAMRolePolicy(awsSession *session.Session, roleName string, policyName string, policyDocument string) error {
	awsSession = createSessionIfNotExists(awsSession)
	svc := iam.New(awsSession)

	getInput := &iam.GetRolePolicyInput{
		PolicyName: aws.String(policyName),
		RoleName:   aws.String(roleName),
	}

	_, err := svc.GetRolePolicy(getInput)
	policyExists := true

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				policyExists = false
			default:
				return err
			}
		} else {
			return err
		}
	}

	if policyExists {
		return nil
	}

	putInput := &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(policyDocument),
		PolicyName:     aws.String(policyName),
		RoleName:       aws.String(roleName),
	}

	_, err = svc.PutRolePolicy(putInput)
	return err
}

// ValidateIAMRolePolicy validates an IAM role inline policy
func ValidateIAMRolePolicy(awsSession *session.Session, roleName string, policyName string) error {
	awsSession = createSessionIfNotExists(awsSession)
	svc := iam.New(awsSession)

	getInput := &iam.GetRolePolicyInput{
		PolicyName: aws.String(policyName),
		RoleName:   aws.String(roleName),
	}

	_, err := svc.GetRolePolicy(getInput)
	policyExists := true

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				policyExists = false
			default:
				return err
			}
		} else {
			return err
		}
	}

	if !policyExists {
		return fmt.Errorf("Role - %s - does not have policy - %s", roleName, policyName)
	}

	return nil
}

// DeleteIAMRolePolicy deletes IAM role policy
func DeleteIAMRolePolicy(awsSession *session.Session, roleName string, policyName string) error {
	awsSession = createSessionIfNotExists(awsSession)
	svc := iam.New(awsSession)

	input := &iam.DeleteRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	}

	_, err := svc.DeleteRolePolicy(input)
	return err
}

// AssumeIAMRole assumes an IAM and returns session and credentials
func AssumeIAMRole(roleArn string, region string) (*session.Session, error) {
	var sess *session.Session
	var creds *credentials.Credentials
	sess, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	if err != nil {
		return sess, err
	}
	creds = stscreds.NewCredentials(sess, roleArn)
	sess.Config.Credentials = creds
	return sess, nil
}

// TrimPolicyDocument converts newlines and spaces into a policyDocument format
// required while creating role
func TrimPolicyDocument(policyDocument string) string {
	return strings.NewReplacer("\n", "", " ", "", "\t", "").Replace(policyDocument)
}

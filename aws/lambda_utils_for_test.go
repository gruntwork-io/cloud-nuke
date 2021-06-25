package aws

import (
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	gruntworkerrors "github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
)

func createLambdaRole(t *testing.T, awsSession *session.Session, roleName string) iam.Role {
	svc := iam.New(awsSession)
	createRoleParams := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: awsgo.String(LAMBDA_ASSUME_ROLE_POLICY),
		RoleName:                 awsgo.String(roleName),
	}

	result, err := svc.CreateRole(createRoleParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	putRolePolicyParams := &iam.PutRolePolicyInput{
		RoleName:       awsgo.String(roleName),
		PolicyDocument: awsgo.String(LAMBDA_ROLE_POLICY),
		PolicyName:     awsgo.String(roleName + "Policy"),
	}
	_, err = svc.PutRolePolicy(putRolePolicyParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	return *result.Role
}

func deleteLambdaRole(awsSession *session.Session, role iam.Role) error {
	svc := iam.New(awsSession)
	listRolePoliciesParams := &iam.ListRolePoliciesInput{
		RoleName: role.RoleName,
	}
	result, err := svc.ListRolePolicies(listRolePoliciesParams)
	if err != nil {
		return gruntworkerrors.WithStackTrace(err)
	}
	for _, policyName := range result.PolicyNames {
		deleteRolePolicyParams := &iam.DeleteRolePolicyInput{
			RoleName:   role.RoleName,
			PolicyName: policyName,
		}
		_, err := svc.DeleteRolePolicy(deleteRolePolicyParams)
		if err != nil {
			return gruntworkerrors.WithStackTrace(err)
		}
	}
	deleteRoleParams := &iam.DeleteRoleInput{
		RoleName: role.RoleName,
	}
	_, err = svc.DeleteRole(deleteRoleParams)
	if err != nil {
		return gruntworkerrors.WithStackTrace(err)
	}
	return nil
}

const LAMBDA_ASSUME_ROLE_POLICY = `{
	"Version": "2012-10-17",
	"Statement": [
	  {
		"Effect": "Allow",
		"Principal": {
		  "Service": "lambda.amazonaws.com"
		},
		"Action": "sts:AssumeRole"
	  }
	]
  }`

const LAMBDA_ROLE_POLICY = `{
	"Version": "2012-10-17",
	"Statement": [
	  {
		"Effect": "Allow",
		"Action": [
			"lambda:CreateFunction",
			"lambda:ListVersionsByFunction",
			"lambda:GetLayerVersion",
			"lambda:PublishLayerVersion",
			"lambda:DeleteProvisionedConcurrencyConfig",
			"lambda:InvokeAsync",
			"lambda:GetAccountSettings",
			"lambda:GetFunctionConfiguration",
			"lambda:GetProvisionedConcurrencyConfig",
			"lambda:ListTags",
			"lambda:DeleteLayerVersion",
			"lambda:PutFunctionEventInvokeConfig",
			"lambda:DeleteFunctionEventInvokeConfig",
			"lambda:DeleteFunction",
			"lambda:ListFunctions",
			"lambda:GetEventSourceMapping",
			"lambda:InvokeFunction",
			"lambda:GetFunction"
		],
		"Resource": [
		  "*"
		]
	  }
	]
  }`

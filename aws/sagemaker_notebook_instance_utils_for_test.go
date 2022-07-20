package aws

import (
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	gruntworkerrors "github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
)

func createNotebookRole(t *testing.T, awsSession *session.Session, roleName string) iam.Role {
	svc := iam.New(awsSession)
	createRoleParams := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: awsgo.String(SAGEMAKER_ASSUME_ROLE_POLICY),
		RoleName:                 awsgo.String(roleName),
	}

	result, err := svc.CreateRole(createRoleParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	putRolePolicyParams := &iam.PutRolePolicyInput{
		RoleName:       awsgo.String(roleName),
		PolicyDocument: awsgo.String(SAGEMAKER_ROLE_POLICY),
		PolicyName:     awsgo.String(roleName + "Policy"),
	}
	_, err = svc.PutRolePolicy(putRolePolicyParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	return *result.Role
}

func deleteNotebookRole(awsSession *session.Session, role iam.Role) error {
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

const SAGEMAKER_ASSUME_ROLE_POLICY = `{
	"Version": "2012-10-17",
	"Statement": [
	  {
		"Effect": "Allow",
		"Principal": {
		  "Service": "sagemaker.amazonaws.com"
		},
		"Action": "sts:AssumeRole"
	  }
	]
  }`

const SAGEMAKER_ROLE_POLICY = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "ec2:CreateTags",
            "Resource": [
                "arn:aws:ec2:*:*:network-interface/*",
                "arn:aws:ec2:*:*:security-group/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:CreateNetworkInterface",
                "ec2:CreateSecurityGroup",
                "ec2:DeleteNetworkInterface",
                "ec2:DescribeDhcpOptions",
                "ec2:DescribeNetworkInterfaces",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeSubnets",
                "ec2:DescribeVpcs",
                "ec2:ModifyNetworkInterfaceAttribute"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:AuthorizeSecurityGroupEgress",
                "ec2:AuthorizeSecurityGroupIngress",
                "ec2:CreateNetworkInterfacePermission",
                "ec2:DeleteNetworkInterfacePermission",
                "ec2:DeleteSecurityGroup",
                "ec2:RevokeSecurityGroupEgress",
                "ec2:RevokeSecurityGroupIngress"
            ],
            "Resource": "*",
            "Condition": {
                "StringLike": {
                    "ec2:ResourceTag/ManagedByAmazonSageMakerResource": "*"
                }
            }
        }
    ]
}`

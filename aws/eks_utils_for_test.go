package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	gruntworkerrors "github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getRandomEksSupportedRegion - Returns a random AWS region that supports EKS.
// Refer to https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/
func getRandomEksSupportedRegion() string {
	supportedRegions := []string{
		"us-east-1",
		"us-east-2",
		"us-west-2",
		"eu-west-1",
	}
	rand.Seed(time.Now().UnixNano())
	randIndex := rand.Intn(len(supportedRegions))
	return supportedRegions[randIndex]
}

func createEksCluster(
	t *testing.T,
	awsSession *session.Session,
	randomId string,
	roleArn string,
) eks.Cluster {
	clusterName := fmt.Sprintf("cloud-nuke-%s-%s", t.Name(), randomId)
	subnet1, subnet2 := getSubnetsInDifferentAZs(t, awsSession)

	svc := eks.New(awsSession)
	result, err := svc.CreateCluster(&eks.CreateClusterInput{
		Name:    awsgo.String(clusterName),
		RoleArn: awsgo.String(roleArn),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SubnetIds: []*string{subnet1.SubnetId, subnet2.SubnetId},
		},
	})
	if err != nil {
		require.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	err = svc.WaitUntilClusterActive(&eks.DescribeClusterInput{Name: result.Cluster.Name})
	if err != nil {
		require.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	return *result.Cluster
}

func createEksClusterRole(
	t *testing.T,
	awsSession *session.Session,
	randomId string,
) iam.Role {
	roleName := fmt.Sprintf("cloud-nuke-%s-%s", t.Name(), randomId)
	svc := iam.New(awsSession)
	createRoleParams := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: awsgo.String(EKS_ASSUME_ROLE_POLICY),
		RoleName:                 awsgo.String(roleName),
	}
	result, err := svc.CreateRole(createRoleParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	attachRolePolicy(t, svc, roleName, "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy")
	attachRolePolicy(t, svc, roleName, "arn:aws:iam::aws:policy/AmazonEKSServicePolicy")

	// IAM resources are slow to propagate, so give it some
	// time
	time.Sleep(15 * time.Second)

	return *result.Role
}

func attachRolePolicy(t *testing.T, svc *iam.IAM, roleName string, policyArn string) {
	attachRolePolicyParams := &iam.AttachRolePolicyInput{
		RoleName:  awsgo.String(roleName),
		PolicyArn: awsgo.String(policyArn),
	}
	_, err := svc.AttachRolePolicy(attachRolePolicyParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
}

const EKS_ASSUME_ROLE_POLICY = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "eks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}`

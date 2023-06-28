package aws

import (
	"fmt"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	gruntworkerrors "github.com/gruntwork-io/go-commons/errors"
	terraws "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type privateSubnet struct {
	routeTableID            *string
	subnetID                *string
	routeTableAssociationID *string
}

func createEKSCluster(
	t *testing.T,
	awsSession *session.Session,
	randomID string,
	roleArn string,
) eks.Cluster {
	clusterName := fmt.Sprintf("cloud-nuke-%s-%s", t.Name(), randomID)
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

func createEKSFargateProfile(
	t *testing.T,
	awsSession *session.Session,
	clusterName *string,
	randomID string,
	podExecutionRoleArn *string,
	subnet privateSubnet,
) {
	name := awsgo.String("cloud-nuke")
	svc := eks.New(awsSession)
	input := &eks.CreateFargateProfileInput{
		ClientRequestToken:  awsgo.String(randomID),
		ClusterName:         clusterName,
		FargateProfileName:  name,
		PodExecutionRoleArn: podExecutionRoleArn,
		Selectors: []*eks.FargateProfileSelector{
			&eks.FargateProfileSelector{Namespace: awsgo.String("default")},
		},
		Subnets: []*string{subnet.subnetID},
	}
	_, err := svc.CreateFargateProfile(input)
	require.NoError(t, err)
	require.NoError(
		t,
		svc.WaitUntilFargateProfileActive(&eks.DescribeFargateProfileInput{
			ClusterName:        clusterName,
			FargateProfileName: name,
		}),
	)
}

func createEKSNodeGroup(
	t *testing.T,
	awsSession *session.Session,
	clusterName *string,
	randomID string,
	nodeGroupRole *string,
) {
	subnet1, subnet2 := getSubnetsInDifferentAZs(t, awsSession)

	name := awsgo.String("cloud-nuke")
	svc := eks.New(awsSession)
	input := &eks.CreateNodegroupInput{
		ClusterName:   clusterName,
		AmiType:       awsgo.String("AL2_x86_64"),
		NodegroupName: name,
		NodeRole:      nodeGroupRole,
		Subnets:       []*string{subnet1.SubnetId, subnet2.SubnetId},
	}
	_, err := svc.CreateNodegroup(input)
	require.NoError(t, err)
	require.NoError(
		t,
		svc.WaitUntilNodegroupActive(&eks.DescribeNodegroupInput{
			ClusterName:   clusterName,
			NodegroupName: name,
		}),
	)
}

func createEKSClusterRole(t *testing.T, awsSession *session.Session, randomID string) iam.Role {
	roleName := fmt.Sprintf("cloud-nuke-%s-%s", t.Name(), randomID)
	svc := iam.New(awsSession)
	createRoleParams := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: awsgo.String(eksAssumeRolePolicy),
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

func createEKSNodeGroupRole(t *testing.T, awsSession *session.Session, randomID string) iam.Role {
	roleName := fmt.Sprintf("cloud-nuke-%s-ng-%s", t.Name(), randomID)
	svc := iam.New(awsSession)
	createRoleParams := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: awsgo.String(eksNodeGroupAssumeRolePolicy),
		RoleName:                 awsgo.String(roleName),
	}
	result, err := svc.CreateRole(createRoleParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	attachRolePolicy(t, svc, roleName, "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy")
	attachRolePolicy(t, svc, roleName, "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy")
	attachRolePolicy(t, svc, roleName, "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly")

	// IAM resources are slow to propagate, so give it some
	// time
	time.Sleep(15 * time.Second)

	return *result.Role
}

func createEKSClusterPodExecutionRole(t *testing.T, awsSession *session.Session, randomID string) iam.Role {
	roleName := fmt.Sprintf("cloud-nuke-%s-fargate-%s", t.Name(), randomID)
	svc := iam.New(awsSession)
	createRoleParams := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: awsgo.String(eksFargateAssumeRolePolicy),
		RoleName:                 awsgo.String(roleName),
	}
	result, err := svc.CreateRole(createRoleParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
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

// createPrivateSubnetE will create a private subnet into the default VPC. To create a private subnet, we need to first
// create a route table that has no internet gateway associated with it. This function will return the RouteTable and
// Subnet that it creates so that it can be deleted later. This returns the error so that we can cleanup any partial
// resources that are created if downstream resources error.
func createPrivateSubnetE(t *testing.T, session *session.Session) (privateSubnet, error) {
	subnet := privateSubnet{}
	svc := ec2.New(session)
	defaultVPC := terraws.GetDefaultVpc(t, awsgo.StringValue(session.Config.Region))

	subnetName := fmt.Sprintf("cloud-nuke-%s", t.Name())
	nameTags := []*ec2.Tag{
		{
			Key:   awsgo.String("Name"),
			Value: awsgo.String(subnetName),
		},
	}

	createRouteTableOutput, err := svc.CreateRouteTable(&ec2.CreateRouteTableInput{
		VpcId: awsgo.String(defaultVPC.Id),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: awsgo.String("route-table"),
				Tags:         nameTags,
			},
		},
	})
	if err != nil {
		return subnet, err
	}
	subnet.routeTableID = createRouteTableOutput.RouteTable.RouteTableId

	createSubnetOutput, err := svc.CreateSubnet(&ec2.CreateSubnetInput{
		CidrBlock: awsgo.String("172.31.173.0/24"),
		VpcId:     awsgo.String(defaultVPC.Id),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: awsgo.String("subnet"),
				Tags:         nameTags,
			},
		},
	})
	if err != nil {
		return subnet, err
	}
	subnet.subnetID = createSubnetOutput.Subnet.SubnetId

	associateRouteTableOutput, err := svc.AssociateRouteTable(&ec2.AssociateRouteTableInput{
		RouteTableId: subnet.routeTableID,
		SubnetId:     subnet.subnetID,
	})
	if err == nil {
		subnet.routeTableAssociationID = associateRouteTableOutput.AssociationId
	}
	return subnet, err
}

// deletePrivateSubnet deletes the subnet and associated private route table for the VPC.
func deletePrivateSubnet(t *testing.T, session *session.Session, subnet privateSubnet) {
	svc := ec2.New(session)
	if subnet.routeTableAssociationID != nil {
		_, err := svc.DisassociateRouteTable(&ec2.DisassociateRouteTableInput{
			AssociationId: subnet.routeTableAssociationID,
		})
		require.NoError(t, err)
	}
	if subnet.subnetID != nil {
		_, err := svc.DeleteSubnet(&ec2.DeleteSubnetInput{
			SubnetId: subnet.subnetID,
		})
		require.NoError(t, err)
	}
	if subnet.routeTableID != nil {
		_, err := svc.DeleteRouteTable(&ec2.DeleteRouteTableInput{
			RouteTableId: subnet.routeTableID,
		})
		require.NoError(t, err)
	}
}

const eksAssumeRolePolicy = `{
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

const eksFargateAssumeRolePolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "eks-fargate-pods.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}`

const eksNodeGroupAssumeRolePolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}`

package aws

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/gruntwork-cli/collections"
	gruntworkerrors "github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gruntwork-io/cloud-nuke/logging"
)

// We black list us-east-1e because this zone is frequently out of capacity
var AvailabilityZoneBlackList = []string{"us-east-1e"}

// getRandomFargateSupportedRegion - Returns a random AWS
// region that supports Fargate.
// Refer to https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/
func getRandomFargateSupportedRegion() string {
	supportedRegions := []string{
		"us-east-1", "us-east-2", "us-west-2",
		"eu-central-1", "eu-west-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
	}
	rand.Seed(time.Now().UnixNano())
	randIndex := rand.Intn(len(supportedRegions))
	return supportedRegions[randIndex]
}

func createEcsFargateCluster(t *testing.T, awsSession *session.Session, name string) ecs.Cluster {
	logging.Logger.Infof("Creating ECS cluster %s in region %s", name, *awsSession.Config.Region)

	svc := ecs.New(awsSession)
	result, err := svc.CreateCluster(&ecs.CreateClusterInput{ClusterName: awsgo.String(name)})
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	return *result.Cluster
}

func createEcsEC2Cluster(t *testing.T, awsSession *session.Session, name string, instanceProfile iam.InstanceProfile) (ecs.Cluster, ec2.Instance) {
	cluster := createEcsFargateCluster(t, awsSession, name)

	ec2Svc := ec2.New(awsSession)
	imageID, err := getAMIIdByName(ec2Svc, "amzn-ami-2018.03.g-amazon-ecs-optimized")
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}

	rawUserDataText := fmt.Sprintf("#!/bin/bash\necho 'ECS_CLUSTER=%s' >> /etc/ecs/ecs.config", *cluster.ClusterName)
	userDataText := base64.StdEncoding.EncodeToString([]byte(rawUserDataText))

	instanceProfileSpecification := &ec2.IamInstanceProfileSpecification{
		Arn: instanceProfile.Arn,
	}
	params := &ec2.RunInstancesInput{
		ImageId:               awsgo.String(imageID),
		InstanceType:          awsgo.String("t3.micro"),
		MinCount:              awsgo.Int64(1),
		MaxCount:              awsgo.Int64(1),
		DisableApiTermination: awsgo.Bool(false),
		IamInstanceProfile:    instanceProfileSpecification,
		UserData:              awsgo.String(userDataText),
	}
	instance, err := runAndWaitForInstance(ec2Svc, name, params)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	// At this point we assume the instance successfully
	// registered itself to the cluster
	return cluster, instance
}

func deleteEcsCluster(awsSession *session.Session, cluster ecs.Cluster) error {
	svc := ecs.New(awsSession)
	params := &ecs.DeleteClusterInput{Cluster: cluster.ClusterArn}
	_, err := svc.DeleteCluster(params)
	if err != nil {
		return gruntworkerrors.WithStackTrace(err)
	}
	return nil
}

func createEcsService(t *testing.T, awsSession *session.Session, serviceName string, cluster ecs.Cluster, launchType string, taskDefinition ecs.TaskDefinition) ecs.Service {
	svc := ecs.New(awsSession)
	createServiceParams := &ecs.CreateServiceInput{
		Cluster:        cluster.ClusterArn,
		DesiredCount:   awsgo.Int64(1),
		LaunchType:     awsgo.String(launchType),
		ServiceName:    awsgo.String(serviceName),
		TaskDefinition: taskDefinition.TaskDefinitionArn,
	}
	if launchType == "FARGATE" {
		vpcConfiguration, err := getVpcConfiguration(awsSession)
		if err != nil {
			assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
		}
		networkConfiguration := &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &vpcConfiguration,
		}
		createServiceParams.SetNetworkConfiguration(networkConfiguration)
	}
	result, err := svc.CreateService(createServiceParams)
	require.NoError(t, err)

	// Wait for the service to come up before continuing. We try at most two times to wait for the service. Oftentimes
	// the service wait times out on the first try, but eventually succeeds.
	retry.DoWithRetry(
		t,
		fmt.Sprintf("Waiting for service %s to be stable", awsgo.StringValue(result.Service.ServiceArn)),
		2,
		0*time.Second,
		func() (string, error) {
			err := svc.WaitUntilServicesStable(&ecs.DescribeServicesInput{
				Cluster:  cluster.ClusterArn,
				Services: []*string{result.Service.ServiceArn},
			})
			return "", err
		},
	)

	return *result.Service
}

func createEcsTaskDefinition(t *testing.T, awsSession *session.Session, taskFamilyName string, launchType string) ecs.TaskDefinition {
	svc := ecs.New(awsSession)
	containerDefinition := &ecs.ContainerDefinition{
		Image: awsgo.String("nginx:latest"),
		Name:  awsgo.String("nginx"),
	}
	registerTaskParams := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{containerDefinition},
		Cpu:                  awsgo.String("256"),
		Memory:               awsgo.String("512"),
		Family:               awsgo.String(taskFamilyName),
	}
	if launchType == "FARGATE" {
		registerTaskParams.SetNetworkMode("awsvpc")
		registerTaskParams.SetRequiresCompatibilities([]*string{awsgo.String("FARGATE")})
	}
	result, err := svc.RegisterTaskDefinition(registerTaskParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	return *result.TaskDefinition
}

func deleteEcsTaskDefinition(awsSession *session.Session, taskDefinition ecs.TaskDefinition) error {
	svc := ecs.New(awsSession)
	deregisterTaskDefinitionParams := &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: taskDefinition.TaskDefinitionArn,
	}
	_, err := svc.DeregisterTaskDefinition(deregisterTaskDefinitionParams)
	if err != nil {
		return gruntworkerrors.WithStackTrace(err)
	}
	return nil
}

func createEcsInstanceProfile(t *testing.T, awsSession *session.Session, instanceProfileName string, role iam.Role) iam.InstanceProfile {
	svc := iam.New(awsSession)
	createInstanceProfileParams := &iam.CreateInstanceProfileInput{
		InstanceProfileName: awsgo.String(instanceProfileName),
	}
	result, err := svc.CreateInstanceProfile(createInstanceProfileParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	instanceProfile := result.InstanceProfile
	addRoleToInstanceProfileParams := &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: instanceProfile.InstanceProfileName,
		RoleName:            role.RoleName,
	}
	_, err = svc.AddRoleToInstanceProfile(addRoleToInstanceProfileParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	return *instanceProfile
}

func deleteInstanceProfile(awsSession *session.Session, instanceProfile iam.InstanceProfile) error {
	svc := iam.New(awsSession)
	getInstanceProfileParams := &iam.GetInstanceProfileInput{
		InstanceProfileName: instanceProfile.InstanceProfileName,
	}
	result, err := svc.GetInstanceProfile(getInstanceProfileParams)
	if err != nil {
		return gruntworkerrors.WithStackTrace(err)
	}
	refreshedInstanceProfile := result.InstanceProfile
	for _, role := range refreshedInstanceProfile.Roles {
		removeRoleParams := &iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: refreshedInstanceProfile.InstanceProfileName,
			RoleName:            role.RoleName,
		}
		_, err := svc.RemoveRoleFromInstanceProfile(removeRoleParams)
		if err != nil {
			return gruntworkerrors.WithStackTrace(err)
		}
	}
	deleteInstanceProfileParams := &iam.DeleteInstanceProfileInput{
		InstanceProfileName: refreshedInstanceProfile.InstanceProfileName,
	}
	_, err = svc.DeleteInstanceProfile(deleteInstanceProfileParams)
	if err != nil {
		return gruntworkerrors.WithStackTrace(err)
	}
	return nil

}

func createEcsRole(t *testing.T, awsSession *session.Session, roleName string) iam.Role {
	svc := iam.New(awsSession)
	createRoleParams := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: awsgo.String(ECS_ASSUME_ROLE_POLICY),
		RoleName:                 awsgo.String(roleName),
	}
	result, err := svc.CreateRole(createRoleParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	putRolePolicyParams := &iam.PutRolePolicyInput{
		RoleName:       awsgo.String(roleName),
		PolicyDocument: awsgo.String(ECS_ROLE_POLICY),
		PolicyName:     awsgo.String(roleName + "Policy"),
	}
	_, err = svc.PutRolePolicy(putRolePolicyParams)
	if err != nil {
		assert.Fail(t, gruntworkerrors.WithStackTrace(err).Error())
	}
	return *result.Role
}

func deleteRole(awsSession *session.Session, role iam.Role) error {
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

func getVpcConfiguration(awsSession *session.Session) (ecs.AwsVpcConfiguration, error) {
	ec2Svc := ec2.New(awsSession)
	describeVpcsParams := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("isDefault"),
				Values: []*string{awsgo.String("true")},
			},
		},
	}
	vpcs, err := ec2Svc.DescribeVpcs(describeVpcsParams)
	if err != nil {
		return ecs.AwsVpcConfiguration{}, gruntworkerrors.WithStackTrace(err)
	}
	if len(vpcs.Vpcs) == 0 {
		err := errors.New(fmt.Sprintf("Could not find any default VPC in region %s", *awsSession.Config.Region))
		return ecs.AwsVpcConfiguration{}, gruntworkerrors.WithStackTrace(err)
	}
	defaultVpc := vpcs.Vpcs[0]

	describeSubnetsParams := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{defaultVpc.VpcId},
			},
		},
	}
	subnets, err := ec2Svc.DescribeSubnets(describeSubnetsParams)
	if err != nil {
		return ecs.AwsVpcConfiguration{}, gruntworkerrors.WithStackTrace(err)
	}
	if len(subnets.Subnets) == 0 {
		err := errors.New(fmt.Sprintf("Could not find any subnets for default VPC in region %s", *awsSession.Config.Region))
		return ecs.AwsVpcConfiguration{}, gruntworkerrors.WithStackTrace(err)
	}
	var subnetIds []*string
	for _, subnet := range subnets.Subnets {
		// Only use public subnets for testing simplicity
		if !collections.ListContainsElement(AvailabilityZoneBlackList, awsgo.StringValue(subnet.AvailabilityZone)) && awsgo.BoolValue(subnet.MapPublicIpOnLaunch) {
			subnetIds = append(subnetIds, subnet.SubnetId)
		}
	}
	vpcConfig := ecs.AwsVpcConfiguration{
		Subnets:        subnetIds,
		AssignPublicIp: awsgo.String(ecs.AssignPublicIpEnabled),
	}
	return vpcConfig, nil
}

const ECS_ASSUME_ROLE_POLICY = `{
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

const ECS_ROLE_POLICY = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:BatchCheckLayerAvailability",
        "ecr:BatchGetImage",
        "ecr:DescribeRepositories",
        "ecr:GetAuthorizationToken",
        "ecr:GetDownloadUrlForLayer",
        "ecr:GetRepositoryPolicy",
        "ecr:ListImages",
        "ecs:CreateCluster",
        "ecs:DeregisterContainerInstance",
        "ecs:DiscoverPollEndpoint",
        "ecs:Poll",
        "ecs:RegisterContainerInstance",
        "ecs:StartTask",
        "ecs:StartTelemetrySession",
        "ecs:SubmitContainerStateChange",
        "ecs:SubmitTaskStateChange"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}`

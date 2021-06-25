package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
)

// Test that we can find ECS services that are running Fargate tasks
func TestListECSFargateServices(t *testing.T) {
	t.Parallel()

	region := getRandomFargateSupportedRegion()
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	ecsServiceClusterMap := map[string]string{}
	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	clusterName := uniqueTestID + "-cluster"
	serviceName := uniqueTestID + "-service"
	taskFamilyName := uniqueTestID + "-task"

	cluster := createEcsFargateCluster(t, awsSession, clusterName)
	defer deleteEcsCluster(awsSession, cluster)

	taskDefinition := createEcsTaskDefinition(t, awsSession, taskFamilyName, "FARGATE")
	defer deleteEcsTaskDefinition(awsSession, taskDefinition)

	service := createEcsService(t, awsSession, serviceName, cluster, "FARGATE", taskDefinition)
	ecsServiceClusterMap[*service.ServiceArn] = *cluster.ClusterArn
	defer nukeAllEcsServices(awsSession, ecsServiceClusterMap, []*string{service.ServiceArn})

	ecsServiceArns, newEcsServiceClusterMap, err := getAllEcsServices(awsSession, []*string{cluster.ClusterArn}, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of services: %s", err.Error())
	}
	assert.NotContains(t, awsgo.StringValueSlice(ecsServiceArns), *service.ServiceArn)
	_, exists := newEcsServiceClusterMap[*service.ServiceArn]
	assert.False(t, exists)

	ecsServiceArns, newEcsServiceClusterMap, err = getAllEcsServices(awsSession, []*string{cluster.ClusterArn}, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of services: %s", err.Error())
	}
	assert.Contains(t, awsgo.StringValueSlice(ecsServiceArns), *service.ServiceArn)
	_, exists = newEcsServiceClusterMap[*service.ServiceArn]
	assert.True(t, exists)
}

// Test that we can successfully nuke ECS services running Fargate tasks
func TestNukeECSFargateServices(t *testing.T) {
	t.Parallel()

	region := getRandomFargateSupportedRegion()
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	clusterName := uniqueTestID + "-cluster"
	serviceName := uniqueTestID + "-service"
	taskFamilyName := uniqueTestID + "-task"

	cluster := createEcsFargateCluster(t, awsSession, clusterName)
	defer deleteEcsCluster(awsSession, cluster)

	taskDefinition := createEcsTaskDefinition(t, awsSession, taskFamilyName, "FARGATE")
	defer deleteEcsTaskDefinition(awsSession, taskDefinition)

	service := createEcsService(t, awsSession, serviceName, cluster, "FARGATE", taskDefinition)

	ecsServiceClusterMap := map[string]string{}
	ecsServiceClusterMap[*service.ServiceArn] = *cluster.ClusterArn
	err = nukeAllEcsServices(awsSession, ecsServiceClusterMap, []*string{service.ServiceArn})
	if err != nil {
		assert.Fail(t, err.Error())
	}

	ecsServiceArns, _, err := getAllEcsServices(awsSession, []*string{cluster.ClusterArn}, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of services: %s", err.Error())
	}
	assert.NotContains(t, awsgo.StringValueSlice(ecsServiceArns), *service.ServiceArn)
}

// Test that we can find ECS services running EC2 tasks
func TestListECSEC2Services(t *testing.T) {
	t.Parallel()

	region := getRandomFargateSupportedRegion()
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	ecsServiceClusterMap := map[string]string{}
	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	clusterName := uniqueTestID + "-cluster"
	serviceName := uniqueTestID + "-service"
	taskFamilyName := uniqueTestID + "-task"
	roleName := uniqueTestID + "-role"
	instanceProfileName := uniqueTestID + "-instance-profile"

	// Prepare resources
	// Create the IAM roles for ECS EC2 container instances
	role := createEcsRole(t, awsSession, roleName)
	defer deleteRole(awsSession, role)

	instanceProfile := createEcsInstanceProfile(t, awsSession, instanceProfileName, role)
	defer deleteInstanceProfile(awsSession, instanceProfile)

	// IAM resources are slow to propagate, so give it some
	// time
	time.Sleep(15 * time.Second)

	// Provision a cluster with ec2 container instances, not
	// forgetting to schedule deletion
	cluster, instance := createEcsEC2Cluster(t, awsSession, clusterName, instanceProfile)
	defer deleteEcsCluster(awsSession, cluster)
	defer nukeAllEc2Instances(awsSession, []*string{instance.InstanceId})

	// Finally, define the task and service
	taskDefinition := createEcsTaskDefinition(t, awsSession, taskFamilyName, "EC2")
	defer deleteEcsTaskDefinition(awsSession, taskDefinition)

	service := createEcsService(t, awsSession, serviceName, cluster, "EC2", taskDefinition)
	ecsServiceClusterMap[*service.ServiceArn] = *cluster.ClusterArn
	defer nukeAllEcsServices(awsSession, ecsServiceClusterMap, []*string{service.ServiceArn})
	// END prepare resources

	ecsServiceArns, newEcsServiceClusterMap, err := getAllEcsServices(awsSession, []*string{cluster.ClusterArn}, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of services: %s", err.Error())
	}
	assert.NotContains(t, awsgo.StringValueSlice(ecsServiceArns), *service.ServiceArn)
	_, exists := newEcsServiceClusterMap[*service.ServiceArn]
	assert.False(t, exists)

	ecsServiceArns, newEcsServiceClusterMap, err = getAllEcsServices(awsSession, []*string{cluster.ClusterArn}, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of services: %s", err.Error())
	}
	assert.Contains(t, awsgo.StringValueSlice(ecsServiceArns), *service.ServiceArn)
	_, exists = newEcsServiceClusterMap[*service.ServiceArn]
	assert.True(t, exists)
}

// Test that we can successfully nuke ECS services running EC2 tasks
func TestNukeECSEC2Services(t *testing.T) {
	t.Parallel()

	region := getRandomFargateSupportedRegion()
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	ecsServiceClusterMap := map[string]string{}
	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	clusterName := uniqueTestID + "-cluster"
	serviceName := uniqueTestID + "-service"
	taskFamilyName := uniqueTestID + "-task"
	roleName := uniqueTestID + "-role"
	instanceProfileName := uniqueTestID + "-instance-profile"

	// Prepare resources
	// Create the IAM roles for ECS EC2 container instances
	role := createEcsRole(t, awsSession, roleName)
	defer deleteRole(awsSession, role)

	instanceProfile := createEcsInstanceProfile(t, awsSession, instanceProfileName, role)
	defer deleteInstanceProfile(awsSession, instanceProfile)

	// IAM resources are slow to propagate, so give it some
	// time
	time.Sleep(15 * time.Second)

	// Provision a cluster with ec2 container instances, not
	// forgetting to schedule deletion
	cluster, instance := createEcsEC2Cluster(t, awsSession, clusterName, instanceProfile)
	defer deleteEcsCluster(awsSession, cluster)
	defer nukeAllEc2Instances(awsSession, []*string{instance.InstanceId})

	// Finally, define the task and service
	taskDefinition := createEcsTaskDefinition(t, awsSession, taskFamilyName, "EC2")
	defer deleteEcsTaskDefinition(awsSession, taskDefinition)

	service := createEcsService(t, awsSession, serviceName, cluster, "EC2", taskDefinition)
	ecsServiceClusterMap[*service.ServiceArn] = *cluster.ClusterArn
	// END prepare resources

	err = nukeAllEcsServices(awsSession, ecsServiceClusterMap, []*string{service.ServiceArn})

	ecsServiceArns, _, err := getAllEcsServices(awsSession, []*string{cluster.ClusterArn}, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of services: %s", err.Error())
	}
	assert.NotContains(t, awsgo.StringValueSlice(ecsServiceArns), *service.ServiceArn)
}

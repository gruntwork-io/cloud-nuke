package aws

import (
	"sync"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that we can successfully list clusters by manually creating a cluster, and then using the list function to find
// it.
func TestListEksClusters(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	uniqueID := util.UniqueID()

	role := createEKSClusterRole(t, awsSession, uniqueID)
	defer deleteRole(awsSession, role)

	cluster := createEKSCluster(t, awsSession, uniqueID, *role.Arn)
	defer nukeAllEksClusters(awsSession, []*string{cluster.Name})

	eksClusterNames, err := getAllEksClusters(awsSession, time.Now().Add(1*time.Hour*-1), config.Config{})
	if err != nil {
		assert.Failf(t, "Unable to fetch list of clusters: %s", err.Error())
	}
	assert.NotContains(t, awsgo.StringValueSlice(eksClusterNames), *cluster.Name)

	eksClusterNames, err = getAllEksClusters(awsSession, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Failf(t, "Unable to fetch list of clusters: %s", err.Error())
	}
	assert.Contains(t, awsgo.StringValueSlice(eksClusterNames), *cluster.Name)
}

// Test that we can successfully nuke EKS clusters by manually creating a cluster, and then using the nuke function to
// delete it.
func TestNukeEksClusters(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	uniqueID := util.UniqueID()

	role := createEKSClusterRole(t, awsSession, uniqueID)
	defer deleteRole(awsSession, role)

	cluster := createEKSCluster(t, awsSession, uniqueID, *role.Arn)
	err = nukeAllEksClusters(awsSession, []*string{cluster.Name})
	require.NoError(t, err)

	eksClusterNames, err := getAllEksClusters(awsSession, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(eksClusterNames), *cluster.Name)
}

// Test that we can successfully nuke EKS clusters with Node Groups and Fargate Profiles.
func TestNukeEksClustersWithCompute(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	uniqueID := util.UniqueID()

	privateSubnet, err := createPrivateSubnetE(t, awsSession)
	defer deletePrivateSubnet(t, awsSession, privateSubnet)
	require.NoError(t, err)
	logger.Logf(t, "Created subnet %s in default VPC", awsgo.StringValue(privateSubnet.subnetID))

	role := createEKSClusterRole(t, awsSession, uniqueID)
	defer deleteRole(awsSession, role)
	podRole := createEKSClusterPodExecutionRole(t, awsSession, uniqueID)
	defer deleteRole(awsSession, podRole)
	nodegroupRole := createEKSNodeGroupRole(t, awsSession, uniqueID)
	defer deleteRole(awsSession, nodegroupRole)

	logger.Logf(t, "Creating test EKS cluster (this may take a while)")
	cluster := createEKSCluster(t, awsSession, uniqueID, awsgo.StringValue(role.Arn))

	// Concurrently provision fargate profile and node group as both take time to provision
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		defer wg.Done()
		logger.Logf(t, "Creating test Fargate Profile (this may take a while)")
		createEKSFargateProfile(t, awsSession, cluster.Name, uniqueID, podRole.Arn, privateSubnet)
	}()
	go func() {
		defer wg.Done()
		logger.Logf(t, "Creating test Node Group (this may take a while)")
		createEKSNodeGroup(t, awsSession, cluster.Name, uniqueID, nodegroupRole.Arn)
	}()
	wg.Wait()

	err = nukeAllEksClusters(awsSession, []*string{cluster.Name})
	require.NoError(t, err)

	eksClusterNames, err := getAllEksClusters(awsSession, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(eksClusterNames), *cluster.Name)
}

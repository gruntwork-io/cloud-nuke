package aws

import (
	"fmt"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that we can succesfully list ECS clusters by manually creating a cluster and then using the list function to find it.
func TestCanCreateAndListEcsCluster(t *testing.T) {
	t.Parallel()

	region := "eu-west-1"
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	clusterName := fmt.Sprintf("test-ecs-cluster-%s", util.UniqueID())
	cluster := createEcsFargateCluster(t, awsSession, clusterName)
	defer deleteEcsCluster(awsSession, cluster)

	clusterArns, err := getAllEcsClusters(awsSession)
	require.NoError(t, err)

	assert.Contains(t, clusterArns, cluster.ClusterArn)
}

// Test we can create a cluster, tag it, and then find the tag
func TestCanTagEcsClusters(t *testing.T) {
	t.Parallel()

	region := "eu-west-1"
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	// create a cluster without tag
	cluster := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster)

	// get cluster
	clusterArns, err := getAllEcsClusters(awsSession)
	require.NoError(t, err)

	//tag with first_seen tag
	for _, clusterArn := range clusterArns {
		tagEcsCluster(awsSession, clusterArn, "first_seen", "today")
		//todo handle errors
	}

	assert.Equal(t, *getClusterTag(awsSession, cluster.ClusterArn, "first_seen"), "first_seen")
}

// Test we can get all ECS clusters younger than < X time based on tags
func TestCanListAllEcsClustersOlderThan24hours(t *testing.T) {
	// create 3 clusters with tags: 1hr, 22hrs, 28hrs
	// get all ecs clusters
	// get tags for each cluster
	// select only clusters older than 24hrs
	// assert return only 1 cluster
}

// Test we can nuke all ECS clusters younger than < X time
func TestCanNukeAllEcsClustersOlderThan24Hours(t *testing.T) {
	// create 3 clusters with tags: 1hr, 25hrs, 28hrs
	// get all ecs clusters
	// get tags for each cluster
	// select only clusters older than 24hrs
	// nuke selected clusters
	// assert 2 clusters nuked
	// assert 1 cluster still left
}

// Test that we can filter ECS clusters by 'created_at' tag value.
func TestCanCreateWithTagAndFilterEcsClustersByTag(t *testing.T) {
	t.Parallel()

	region := "eu-west-1"
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	clusterName := "test-ina-sep-2-with-tag"
	cluster := createEcsFargateCluster(t, awsSession, clusterName) //TODO - also have a separate function to do this so I can test it.
	tagKey := "created_at"
	tagValue := "now-inastime"
	tag, err := tagEcsCluster(awsSession, cluster.ClusterArn, tagKey, tagValue) //should return
	require.NoError(t, err)
	fmt.Println(tag)
	// TO UNCOMMENT defer deleteEcsCluster(awsSession, cluster)

	// OLD - assert.Contains(t, clusterArns, cluster.ClusterArn)

	// assert that cluster was created with tags
	// assert.True(t, tagIsPresent(awsSession, cluster.ClusterArn))
	//clusterArns, err := getAllEcsClusters(awsSession)
	// filter results - possibly using api
	// assert that result contains your new cluster only (provided it's the only one created in the short time frame)
}

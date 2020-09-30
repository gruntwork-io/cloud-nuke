package aws

import (
	"fmt"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

// Test that we can succesfully list ECS clusters by manually creating a cluster and then using the list function to find it.
func TestCanCreateAndListEcsCluster(t *testing.T) {
	t.Parallel()

	region := "eu-west-1"
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	clusterName := "test-ina-sep-1"
	cluster := createEcsFargateCluster(t, awsSession, clusterName)
	defer deleteEcsCluster(awsSession, cluster)

	clusterArns, err := getAllEcsClusters(awsSession)
	if err != nil {
		assert.Failf(t, "Unable to fetch clusters: %s", err.Error())
	}

	assert.Contains(t, clusterArns, cluster.ClusterArn)
}

// Test we can create a cluster, tag it, and then find the cluster
func TestCanTagEcsClusterAndFilterByTag(t *testing.T) {
	// create a cluster without tag
	// get cluster
	// tag with "first_seen" : "current timestamp"
	// get cluster by tag
	// assert pass
}

// Test we can create a tag and set it to ECS clusters without 'created_at' tags
func TestWontTagEcsClusterWithTag(t *testing.T) {
	// create a cluster with tag
	// get cluster
	// try tag with "first_seen" : "current timestamp"
	// assert fail
}

// Test we can get all ECS clusters younger than < X time based on tags

// Test we can nuke all ECS clusters younger than < X time

// Test that we can filter ECS clusters by 'created_at' tag value.
func TestCanCreateWithTagAndFilterEcsClustersByTag(t *testing.T) {
	t.Parallel()

	region := "eu-west-1"
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	clusterName := "test-ina-sep-2-with-tag"
	cluster := createEcsFargateCluster(t, awsSession, clusterName) //TODO - also have a separate function to do this so I can test it.
	tagKey := "created_at"
	tagValue := "now-inastime"
	tag, err := tagEcsCluster(awsSession, cluster.ClusterArn, tagKey, tagValue) //should return
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	fmt.Println(tag)
	// TO UNCOMMENT defer deleteEcsCluster(awsSession, cluster)

	// OLD - assert.Contains(t, clusterArns, cluster.ClusterArn)

	// assert that cluster was created with tags
	assert.True(t, tagIsPresent(awsSession, cluster.ClusterArn))
	//clusterArns, err := getAllEcsClusters(awsSession)
	// filter results - possibly using api
	// assert that result contains your new cluster only (provided it's the only one created in the short time frame)
}

// Test that we can delete ECS clusters by manually creating an ECS cluster, and then deleting it using the nuke function.
func TestNukeEcsClusters(t *testing.T) {
	// create ECS cluster
	// list all ECS clusters
	// nuke all ECS clusters
	// list all ECS clusters
	// assert no ECS clusters left
}

// Test that we can delete tagged ECS clusters by manually creating an ECS cluster, and then deleting it using the nuke function.
func TestNukeEcsClustersByTag(t *testing.T) {
	// create ECS cluster with a 'created_at' tag
	// list all ECS clusters with this tag & filter
	// nuke ECS cluster by tag and filter
	// list all ECS clusters
	// assert no ECS clusters as filter criteria
}

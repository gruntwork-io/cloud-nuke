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
func TestListEcsClusters(t *testing.T) {
	// create ECS cluster in a specific region
	t.Parallel()

	region := "eu-west-1"
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	clusterName := "test-ina-2"
	cluster := createEcsFargateCluster(t, awsSession, clusterName)
	defer deleteEcsCluster(awsSession, cluster)
	//END prepare resources

	// call api to list all clusters
	clusterArns, err := getAllEcsClusters(awsSession)
	if err != nil {
		assert.Failf(t, "Unable to fetch clusters: %s", err.Error())
	}

	// assert that result contains your new cluster
	assert.Contains(t, clusterArns, cluster.ClusterArn)
}

// Test that we can filter ECS clusters by 'created_at' tag value.
func TestFilterEcsClusters(t *testing.T) {
	// create ECS cluster in a specific region
	// tag that ECS cluster
	t.Parallel()

	region := "eu-west-1"
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	clusterName := "test-ina-2"
	cluster := createEcsFargateCluster(t, awsSession, clusterName)

	tagKey := "created_at"
	tagValue := "now"
	tag := tagEcsCluster(t, awsSession, cluster.ClusterArn, tagKey, tagValue)

	fmt.Println(tagKey)
	//defer deleteEcsCluster(awsSession, cluster)
	//END prepare resources

	// call api to list all clusters
	clusterArns, err := getAllEcsClusters(awsSession)
	if err != nil {
		assert.Failf(t, "Unable to fetch clusters: %s", err.Error())
	}
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

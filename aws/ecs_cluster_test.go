package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tagKey = "first_seen"
const region = "eu-west-1"

// Test we can create a cluster, tag it, and then find the tag
func TestCanTagEcsClusters(t *testing.T) {
	t.Parallel()

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	cluster := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster)

	tagValue := time.Now().UTC().Format(time.RFC3339)

	tagEcsCluster(awsSession, cluster.ClusterArn, tagKey, tagValue)
	require.NoError(t, err)

	returnedTag, err := getClusterTag(awsSession, cluster.ClusterArn, tagKey)
	require.NoError(t, err)

	assert.Equal(t, returnedTag.Format(time.RFC3339), tagValue)
}

// Test we can get all ECS clusters younger than < X time based on tags
func TestCanListAllEcsClustersOlderThan24hours(t *testing.T) {
	t.Parallel()

	region := "eu-west-1"
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	cluster1 := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster1)
	cluster2 := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster2)

	now := time.Now().UTC()
	var olderClusterTagValue = now.Add(time.Hour * time.Duration(-48)).Format(time.RFC3339)
	var youngerClusterTagValue = now.Add(time.Hour * time.Duration(-23)).Format(time.RFC3339)

	tagEcsCluster(awsSession, cluster1.ClusterArn, tagKey, olderClusterTagValue)
	tagEcsCluster(awsSession, cluster2.ClusterArn, tagKey, youngerClusterTagValue)
	require.NoError(t, err)

	last24Hours := now.Add(time.Hour * time.Duration(-24))
	filteredClusterArns, err := getAllEcsClustersOlderThan(awsSession, region, last24Hours)
	require.NoError(t, err)

	assert.Equal(t, 1, len(filteredClusterArns))
}

// Test we can nuke all ECS clusters older than 24hrs
func TestCanNukeAllEcsClustersOlderThan24Hours(t *testing.T) {
	t.Parallel()

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	cluster1 := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster1)
	cluster2 := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster2)
	cluster3 := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster3)

	now := time.Now().UTC()
	var oldClusterTagValue1 = now.Add(time.Hour * time.Duration(-48)).Format(time.RFC3339)
	var youngClusterTagValue = now.Format(time.RFC3339)
	var oldClusterTagValue2 = now.Add(time.Hour * time.Duration(-27)).Format(time.RFC3339)

	tagEcsCluster(awsSession, cluster1.ClusterArn, tagKey, oldClusterTagValue1)
	tagEcsCluster(awsSession, cluster2.ClusterArn, tagKey, youngClusterTagValue)
	tagEcsCluster(awsSession, cluster3.ClusterArn, tagKey, oldClusterTagValue2)
	require.NoError(t, err)

	last24Hours := now.Add(time.Hour * time.Duration(-24))
	filteredClusterArns, err := getAllEcsClustersOlderThan(awsSession, region, last24Hours)
	require.NoError(t, err)

	nukeEcsClusters(awsSession, filteredClusterArns)
	require.NoError(t, err)

	allLeftClusterArns, err := getAllEcsClusters(awsSession)
	require.NoError(t, err)
	assert.Equal(t, 1, len(allLeftClusterArns))
}

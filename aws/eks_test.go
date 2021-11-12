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

// Test that we can successfully list clusters by manually creating a cluster, and then using the list function to find
// it.
func TestListEksClusters(t *testing.T) {
	t.Parallel()

	region := getRandomEksSupportedRegion(t)
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	uniqueID := util.UniqueID()

	role := createEksClusterRole(t, awsSession, uniqueID)
	defer deleteRole(awsSession, role)

	cluster := createEksCluster(t, awsSession, uniqueID, *role.Arn)
	defer nukeAllEksClusters(awsSession, []*string{cluster.Name})

	eksClusterNames, err := getAllEksClusters(awsSession, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of clusters: %s", err.Error())
	}
	assert.NotContains(t, awsgo.StringValueSlice(eksClusterNames), *cluster.Name)

	eksClusterNames, err = getAllEksClusters(awsSession, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of clusters: %s", err.Error())
	}
	assert.Contains(t, awsgo.StringValueSlice(eksClusterNames), *cluster.Name)
}

// Test that we can successfully nuke EKS clusters by manually creating a cluster, and then using the nuke function to
// delete it.
func TestNukeEksClusters(t *testing.T) {
	t.Parallel()

	region := getRandomEksSupportedRegion(t)
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	uniqueID := util.UniqueID()

	role := createEksClusterRole(t, awsSession, uniqueID)
	defer deleteRole(awsSession, role)

	cluster := createEksCluster(t, awsSession, uniqueID, *role.Arn)
	err = nukeAllEksClusters(awsSession, []*string{cluster.Name})
	require.NoError(t, err)

	eksClusterNames, err := getAllEksClusters(awsSession, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(eksClusterNames), *cluster.Name)
}

package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that we can successfully list clusters by manually creating a cluster, and then using the list function to find
// it.
func TestListEksClusters(t *testing.T) {
	t.Parallel()

	region := getRandomEksSupportedRegion()
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	if err != nil {
		require.Fail(t, errors.WithStackTrace(err).Error())
	}

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

	region := getRandomEksSupportedRegion()
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueID := util.UniqueID()

	role := createEksClusterRole(t, awsSession, uniqueID)
	defer deleteRole(awsSession, role)

	cluster := createEksCluster(t, awsSession, uniqueID, *role.Arn)
	err = nukeAllEksClusters(awsSession, []*string{cluster.Name})
	if err != nil {
		assert.Fail(t, err.Error())
	}

	eksClusterNames, err := getAllEksClusters(awsSession, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Failf(t, "Unable to fetch list of clusters: %s", err.Error())
	}
	assert.NotContains(t, awsgo.StringValueSlice(eksClusterNames), *cluster.Name)
}

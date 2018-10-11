package aws

import (
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

// Test that we can find ECS clusters
func TestListECSClusters(t *testing.T) {
	t.Parallel()

	region := getRandomFargateSupportedRegion()
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	clusterName := uniqueTestID + "-cluster"
	cluster := createEcsFargateCluster(t, awsSession, clusterName)
	defer nukeAllEcsClusters(awsSession, []*string{cluster.ClusterArn})

	clusterArns, err := getAllEcsClusters(awsSession)
	if err != nil {
		assert.Failf(t, "Unable to fetch clusters: %s", err.Error())
	}
	assert.Contains(t, clusterArns, cluster.ClusterArn)
}

// Test that we can successfully nuke ECS clusters
func TestNukeECSClusters(t *testing.T) {
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

	cluster := createEcsFargateCluster(t, awsSession, clusterName)
	if err := nukeAllEcsClusters(awsSession, []*string{cluster.ClusterArn}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	clusterArns, err := getAllEcsClusters(awsSession)
	if err != nil {
		assert.Failf(t, "Unable to fetch clusters: %s", err.Error())
	}
	assert.NotContains(t, clusterArns, cluster.ClusterArn)
}

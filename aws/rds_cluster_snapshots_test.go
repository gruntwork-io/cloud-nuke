package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Custom waiter function waitUntilRdsClusterAvailable uses the Amazon RDS API operation DescribeDBClustersInput
// to wait for a condition to be met before returning.
// If the condition is not met within the max attempt window, an error will be returned.
func waitUntilRdsClusterAvailable(svc *rds.RDS, name *string) error {
	input := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: name,
	}

	for i := 0; i < 90; i++ {
		_, err := svc.DescribeDBClusters(input)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Cluster to be created")
	}

	return RdsClusterAvailableError{name: *name}
}

// Built-in waiter function WaitUntilDBClusterSnapshotAvailable not working as expected.
// Created a custom one
func waitUntilRdsClusterSnapshotAvailable(svc *rds.RDS, clusterName *string, snapshotName *string) error {
	input := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterIdentifier:         clusterName,
		DBClusterSnapshotIdentifier: snapshotName,
	}
	for i := 0; i < 90; i++ {
		_, err := svc.DescribeDBClusterSnapshots(input)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Cluster Snapshot to be created")
	}

	return RdsClusterSnapshotAvailableError{clusterName: *clusterName, snapshotName: *snapshotName}
}

func createTestDBCluster(t *testing.T, session *session.Session, name string) {
	svc := rds.New(session)
	input := &rds.CreateDBClusterInput{
		DBClusterIdentifier: awsgo.String(name),
		Engine:              awsgo.String("aurora-mysql"),
		MasterUsername:      awsgo.String("gruntwork"),
		MasterUserPassword:  awsgo.String("password"),
	}

	_, err := svc.CreateDBCluster(input)
	require.NoError(t, err)
	waitUntilRdsClusterAvailable(svc, &name)

}

func createTestRDSClusterSnapshot(t *testing.T, session *session.Session, clusterName string, snapshotName string) {
	svc := rds.New(session)
	input := &rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         awsgo.String(clusterName),
		DBClusterSnapshotIdentifier: awsgo.String(snapshotName),
	}

	_, err := svc.CreateDBClusterSnapshot(input)
	require.NoError(t, errors.WithStackTrace(err))
	waitUntilRdsClusterSnapshotAvailable(svc, &clusterName, &snapshotName)

}

func TestNukeRDSClusterSnapshot(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	snapshotName := "cloud-nuke-test-" + util.UniqueID()
	clusterName := "cloud-nuke-test-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)

	createTestDBCluster(t, session, clusterName)
	clusterNames, err := getAllRdsClusters(session, excludeAfter)
	clusterIdentifier := awsgo.StringValueSlice(clusterNames)[0]
	createTestRDSClusterSnapshot(t, session, clusterIdentifier, snapshotName)

	defer func() {
		nukeAllRdsClusterSnapshots(session, []*string{&snapshotName})

		snapshotNames, _ := getAllRdsClusterSnapshots(session, excludeAfter)

		assert.NotContains(t, awsgo.StringValueSlice(snapshotNames), strings.ToLower(snapshotName))

		//clean up DB Cluster created with createTestDBCluster
		nukeAllRdsClusters(session, []*string{&clusterName})
	}()

	snapShots, err := getAllRdsClusterSnapshots(session, excludeAfter)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Cluster snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotName))

}

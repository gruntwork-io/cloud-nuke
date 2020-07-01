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

func waitUntilRdsClusterAvailable(svc *rds.RDS, name *string) error {
	input := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: name,
	}

	logging.Logger.Info("Waiting for RDS DB Cluster to be created...")
	for i := 0; i < 90; i++ {
		_, err := svc.DescribeDBClusters(input)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}

	return RdsClusterAvailableError{name: *name}
}

func waitUntilRdsClusterSnapshotAvailable(svc *rds.RDS, clusterName *string, snapshotName *string) error {
	input := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterIdentifier: clusterName,
		DBClusterSnapshotIdentifier: snapshotName,
	}
    logging.Logger.Info("Waiting for RDS DB Cluster Snapshot to be created...")
	for i := 0; i < 90; i++ {
		_, err := svc.DescribeDBClusterSnapshots(input)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}

	return RdsClusterSnapshotAvailableError{clusterName: *clusterName, snapshotName: *snapshotName}
}

func createTestDBCluster(t *testing.T, session *session.Session, name string) {
	svc := rds.New(session)
	params := &rds.CreateDBClusterInput{
		DBClusterIdentifier: awsgo.String(name),
		Engine:              awsgo.String("aurora-mysql"),
		MasterUsername:      awsgo.String("gruntwork"),
		MasterUserPassword:  awsgo.String("password"),
	}

	_, err := svc.CreateDBCluster(params)
	require.NoError(t, err)
	waitUntilRdsClusterAvailable(svc, &name)

}

func createTestRDSClusterSnapShot(t *testing.T, session *session.Session, clusterName string, snapshotName string) {
	svc := rds.New(session)
	params := &rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         awsgo.String(clusterName),
		DBClusterSnapshotIdentifier: awsgo.String(snapshotName),
	}

	_, err := svc.CreateDBClusterSnapshot(params)
	require.NoError(t, errors.WithStackTrace(err))
    waitUntilRdsClusterSnapshotAvailable(svc, &clusterName, &snapshotName)

}

func TestNukeRDSClusterSnapShot(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	snapShotName := "cloud-nuke-test-" + util.UniqueID()
	clusterName := "cloud-nuke-test-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)

	createTestDBCluster(t, session, clusterName)
	clusterNames, err := getAllRdsClusters(session, excludeAfter)
	clusterIdentifier := awsgo.StringValueSlice(clusterNames)[0]
	createTestRDSClusterSnapShot(t, session, clusterIdentifier, snapShotName)

	defer func() {
		nukeAllRdsClusterSnapshots(session, []*string{&snapShotName})

		snapShotNames, _ := getAllRdsClusterSnapshots(session, excludeAfter)

		assert.NotContains(t, awsgo.StringValueSlice(snapShotNames), strings.ToLower(snapShotName))
	}()

	snapShots, err := getAllRdsClusterSnapshots(session, excludeAfter)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Cluster snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapShotName))

}

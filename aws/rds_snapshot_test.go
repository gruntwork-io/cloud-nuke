package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Built-in waiter function WaitUntilDBInstanceAvailable not working as expected.
// Created a custom one
func waitUntilRdsInstanceAvailable(svc *rds.RDS, name *string) error {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: name,
	}

	for i := 0; i < 300; i++ {
		instance, err := svc.DescribeDBInstances(input)
		status := instance.DBInstances[0].DBInstanceStatus

		// If SkipFinalSnapshot = false on delete, should also wait for "backing-up" also to finish
		if awsgo.StringValue(status) != "creating" {
			return nil
		}

		if err != nil {
			return err
		}

		time.Sleep(-1 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Instance to be created")
	}

	return RdsInstanceAvailableError{name: *name}
}

// Built-in waiter function WaitUntilDBSnapshotAvailable not working as expected.
// Created a custom one
func waitUntilRdsSnapshotAvailable(svc *rds.RDS, instanceName *string, snapshotName *string) error {
	input := &rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: instanceName,
		DBSnapshotIdentifier: snapshotName,
	}
	for i := 0; i < 90; i++ {
		_, err := svc.DescribeDBSnapshots(input)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Instance Snapshot to be created")
	}

	return RdsInstanceSnapshotAvailableError{instanceName: *instanceName, snapshotName: *snapshotName}
}

// createTestDBInstance generates a test DBInstance since snapshots can only be created
// from existing DB Instances
func createTestDBInstance(t *testing.T, session *session.Session, name string) {
	svc := rds.New(session)
	input := &rds.CreateDBInstanceInput{
		AllocatedStorage:     awsgo.Int64(5),
		DBInstanceClass:      awsgo.String("db.m5.large"),
		DBInstanceIdentifier: awsgo.String(name),
		Engine:               awsgo.String("postgres"),
		MasterUsername:       awsgo.String("gruntwork"),
		MasterUserPassword:   awsgo.String("password"),
	}

	_, err := svc.CreateDBInstance(input)
	require.NoError(t, err)

	waitUntilRdsInstanceAvailable(svc, &name)
}

//createTestRDSSnapshot generates a test DB Snapshot
func createTestRDSSnapshot(t *testing.T, session *session.Session, instanceName string, snapshotName string) {
	svc := rds.New(session)
	input := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: awsgo.String(instanceName),
		DBSnapshotIdentifier: awsgo.String(snapshotName),
	}

	_, err := svc.CreateDBSnapshot(input)
	require.NoError(t, errors.WithStackTrace(err))

	waitUntilRdsSnapshotAvailable(svc, &instanceName, &snapshotName)
}

//TestNukeIncludedNameRDSSnapshot tests for nuking included names regex in config file
func TestNukeIncludedNameRDSSnapshot(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	snapshotName := "cloud-nuke-test-" + util.UniqueID()
	instanceName := "cloud-nuke-test-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)

	createTestDBInstance(t, session, instanceName)
	instanceNames, err := getAllRdsInstances(session, excludeAfter)
	instanceIdentifier := awsgo.StringValueSlice(instanceNames)[0]
	createTestRDSSnapshot(t, session, instanceIdentifier, snapshotName)

	var configObj *config.Config
	configObj, err = config.GetConfig("../config/mocks/rdsSnapshots_include_names.yaml")

	defer func() {
		nukeAllRdsSnapshots(session, []*string{&snapshotName})

		snapshotNames, _ := getAllRdsSnapshots(session, excludeAfter, *configObj)

		assert.NotContains(t, awsgo.StringValueSlice(snapshotNames), strings.ToLower(snapshotName))

		//Clean up DB Instance created with createTestDBInstance
		nukeAllRdsInstances(session, []*string{&instanceName})

	}()

	snapShots, err := getAllRdsSnapshots(session, excludeAfter, *configObj)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotName))

}

//TestExcludedNameRDSSnapshot tests for excluding snapshots in config file when nuking
func TestExcludedNameRDSSnapshot(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	// Snapshot to nuke
	snapshotName := "cloud-nuke-test-" + util.UniqueID()
	instanceName := "cloud-nuke-test-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)

	createTestDBInstance(t, session, instanceName)
	instanceNames, err := getAllRdsInstances(session, excludeAfter)
	instanceIdentifier := awsgo.StringValueSlice(instanceNames)[0]
	createTestRDSSnapshot(t, session, instanceIdentifier, snapshotName)

	// Snapshot to exclude when nuking
	snapshotExcludedName := "cloud-exclude-snapshot-" + util.UniqueID()
	createTestRDSSnapshot(t, session, instanceIdentifier, snapshotExcludedName)

	var configObj *config.Config
	configObj, err = config.GetConfig("../config/mocks/rdsSnapshots_exclude_names.yaml")

	defer func() {
		nukeAllRdsSnapshots(session, []*string{&snapshotName})

		snapshotNames, _ := getAllRdsSnapshots(session, excludeAfter, *configObj)

		assert.NotContains(t, awsgo.StringValueSlice(snapshotNames), strings.ToLower(snapshotName))

		//Clean up DB Instance created with createTestDBInstance
		nukeAllRdsInstances(session, []*string{&instanceName})

	}()

	snapShots, err := getAllRdsSnapshots(session, excludeAfter, *configObj)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotName))
	assert.NotContains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotExcludedName))

}

// TestFilterRDSSnapshot tests for filtering included and exclude snapshots in config file when nuking
func TestFilterRDSSnapshot(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	// Snapshot to nuke
	snapshotName := "cloud-nuke-test-" + util.UniqueID()
	instanceName := "cloud-nuke-test-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)

	createTestDBInstance(t, session, instanceName)
	instanceNames, err := getAllRdsInstances(session, excludeAfter)
	instanceIdentifier := awsgo.StringValueSlice(instanceNames)[0]
	createTestRDSSnapshot(t, session, instanceIdentifier, snapshotName)

	// Snapshot to exclude when nuking
	snapshotExcludedName := "cloud-exclude-snapshot-" + util.UniqueID()
	createTestRDSSnapshot(t, session, instanceIdentifier, snapshotExcludedName)

	var configObj *config.Config
	configObj, err = config.GetConfig("../config/mocks/rdsSnapshots_filter_names.yaml")

	defer func() {
		snapShots, err := getAllRdsSnapshots(session, excludeAfter, *configObj)

		if err != nil {
			assert.Failf(t, "Unable to fetch list of RDS DB Instance snapshots", errors.WithStackTrace(err).Error())
		}
		nukeAllRdsSnapshots(session, snapShots)

		snapshotNames, _ := getAllRdsSnapshots(session, excludeAfter, *configObj)

		assert.NotContains(t, awsgo.StringValueSlice(snapshotNames), strings.ToLower(snapshotName))

		//Clean up DB Instance created with createTestDBInstance
		nukeAllRdsInstances(session, []*string{&instanceName})
	}()

	snapShots, err := getAllRdsSnapshots(session, excludeAfter, *configObj)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB Instance snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotName))
	assert.NotContains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotExcludedName))

}

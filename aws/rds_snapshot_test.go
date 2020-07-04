package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	svc.WaitUntilDBInstanceAvailable(&rds.DescribeDBInstancesInput{})
}

func createTestRDSSnapshot(t *testing.T, session *session.Session, instanceName string, snapshotName string) {
	svc := rds.New(session)
	input := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: awsgo.String(instanceName),
		DBSnapshotIdentifier: awsgo.String(snapshotName),
	}

	_, err := svc.CreateDBSnapshot(input)
	require.NoError(t, errors.WithStackTrace(err))

	svc.WaitUntilDBSnapshotAvailable(&rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: awsgo.String(instanceName),
		DBSnapshotIdentifier: awsgo.String(snapshotName),
	})
}

func TestNukeRDSSnapshot(t *testing.T) {
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

	}()

	snapShots, err := getAllRdsSnapshots(session, excludeAfter, *configObj)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapshotName))

}

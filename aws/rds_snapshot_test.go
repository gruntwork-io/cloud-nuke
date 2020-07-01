package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestDBInstance(t *testing.T, session *session.Session, name string) {
	svc := rds.New(session)
	params := &rds.CreateDBInstanceInput{
		AllocatedStorage:     awsgo.Int64(5),
		DBInstanceClass:      awsgo.String("db.m5.large"),
		DBInstanceIdentifier: awsgo.String(name),
		Engine:               awsgo.String("postgres"),
		MasterUsername:       awsgo.String("gruntwork"),
		MasterUserPassword:   awsgo.String("password"),
	}

	_, err := svc.CreateDBInstance(params)
	require.NoError(t, err)
    
	svc.WaitUntilDBInstanceAvailable(&rds.DescribeDBInstancesInput{})
}

func createTestRDSSnapShot(t *testing.T, session *session.Session, instanceName string, snapShotName string) {
	svc := rds.New(session)
	params := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: awsgo.String(instanceName),
		DBSnapshotIdentifier: awsgo.String(snapShotName),
	}

	_, err := svc.CreateDBSnapshot(params)
	require.NoError(t, errors.WithStackTrace(err))

	svc.WaitUntilDBSnapshotAvailable(&rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: awsgo.String(instanceName),
		DBSnapshotIdentifier: awsgo.String(snapShotName),
	})
}

func TestNukeRDSSnapShot(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))
    
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	snapShotName := "cloud-nuke-test-" + util.UniqueID()
	instanceName := "cloud-nuke-test-" + util.UniqueID()
	excludeAfter := time.Now().Add(1 * time.Hour)
	
	createTestRDSInstance(t, session, instanceName)
	instanceNames, err := getAllRdsInstances(session, excludeAfter)
	instanceIdentifier := awsgo.StringValueSlice(instanceNames)[0]
	createTestRDSSnapShot(t, session, instanceIdentifier, snapShotName)

	defer func() {
		nukeAllRdsSnapshots(session, []*string{&snapShotName})

		snapShotNames, _ := getAllRdsSnapshots(session, excludeAfter)

		assert.NotContains(t, awsgo.StringValueSlice(snapShotNames), strings.ToLower(snapShotName))
	}()

	snapShots, err := getAllRdsSnapshots(session, excludeAfter)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS DB snapshots", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(snapShots), strings.ToLower(snapShotName))

}

package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestSnapshot(t *testing.T, session *session.Session, name string) ec2.Snapshot {
	svc := ec2.New(session)

	az := awsgo.StringValue(session.Config.Region) + "a"
	volume := createTestEBSVolume(t, session, name, az)
	snapshot, err := svc.CreateSnapshot(&ec2.CreateSnapshotInput{
		VolumeId: volume.VolumeId,
	})

	if err != nil {
		assert.Failf(t, "Could not create test Snapshot", errors.WithStackTrace(err).Error())
	}

	err = svc.WaitUntilSnapshotCompleted(&ec2.DescribeSnapshotsInput{
		OwnerIds:    []*string{awsgo.String("self")},
		SnapshotIds: []*string{snapshot.SnapshotId},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	return *snapshot
}

func TestListSnapshots(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	snapshot := createTestSnapshot(t, session, uniqueTestID)

	// clean up after this test
	defer nukeAllSnapshots(session, []*string{snapshot.SnapshotId})
	defer nukeAllEbsVolumes(session, findEBSVolumesByNameTag(t, session, uniqueTestID))

	snapshots, err := getAllSnapshots(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Snapshots")
	}

	assert.NotContains(t, awsgo.StringValueSlice(snapshots), *snapshot.SnapshotId)

	snapshots, err = getAllSnapshots(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Snapshots")
	}

	assert.Contains(t, awsgo.StringValueSlice(snapshots), *snapshot.SnapshotId)
}

func TestNukeSnapshots(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	svc := ec2.New(session)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	snapshot := createTestSnapshot(t, session, uniqueTestID)

	// clean up ec2 instance created by the above call
	defer nukeAllEbsVolumes(session, findEBSVolumesByNameTag(t, session, uniqueTestID))

	_, err = svc.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{snapshot.SnapshotId},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if err := nukeAllSnapshots(session, []*string{snapshot.SnapshotId}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	snapshots, err := getAllSnapshots(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Snapshots")
	}

	assert.NotContains(t, awsgo.StringValueSlice(snapshots), *snapshot.SnapshotId)
}

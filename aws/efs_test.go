package aws

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/retry"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findEFSVolumesByNameTag(t *testing.T, session *session.Session, name string) []*string {
	output, err := efs.New(session).DescribeFileSystems(&efs.DescribeFileSystemsInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	var volumeIds []*string
	for _, volume := range output.FileSystems {
		if aws.StringValue(volume.Name) == name {
			volumeIds = append(volumeIds, volume.FileSystemId)
		}
	}

	return volumeIds
}

func createTestEFSVolumeMountTarget(t *testing.T, session *session.Session, fileSystemId *string, subnetId *string) efs.MountTargetDescription {
	svc := efs.New(session)

	mountTarget, err := svc.CreateMountTarget(&efs.CreateMountTargetInput{
		FileSystemId: fileSystemId,
		SubnetId: subnetId,
	})

	if err != nil {
		assert.Failf(t, "Could not create test EFS mount target for volume", errors.WithStackTrace(err).Error())
	}

	return *mountTarget
}

func createTestEFSVolume(t *testing.T, session *session.Session, name string, az string) efs.FileSystemDescription {
	svc := efs.New(session)

	volume, err := svc.CreateFileSystem(&efs.CreateFileSystemInput{
		AvailabilityZoneName: aws.String(az),
		Tags: []*efs.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(name),
			},
		},
	})

	if err != nil {
		assert.Failf(t, "Could not create test EFS volume", errors.WithStackTrace(err).Error())
	}

	if err := retry.DoWithRetry(
		logging.Logger,
		fmt.Sprintf("Waiting until EFS volume ID %s is fully created", *volume.FileSystemId),
		10,
		1*time.Second,
		func () error {
			details, err := svc.DescribeFileSystems(&efs.DescribeFileSystemsInput{
				FileSystemId: volume.FileSystemId,
			})
			if err != nil {
				return err
			}
			if len(details.FileSystems) <= 0 || len(details.FileSystems) >= 2 {
				return fmt.Errorf("Something went wrong searching for the volume")
			}
			if aws.StringValue(details.FileSystems[0].LifeCycleState) == "creating" {
				return fmt.Errorf("EFS volume is still creating")
			}
			return nil
		},
	); err != nil {
		assert.Failf(t, "Could not create test EFS volume (%s)", errors.WithStackTrace(err).Error())
	}

	return *volume
}

func TestListEFSVolumes(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	az := aws.StringValue(session.Config.Region) + "a"
	volume := createTestEFSVolume(t, session, uniqueTestID, az)
	defer nukeAllEfsVolumes(session, []*string{volume.FileSystemId})

	volumeIds, err := getAllEfsVolumes(session, time.Now().Add(1*time.Hour*-1), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EFS Volumes")
	}

	assert.NotContains(t, aws.StringValueSlice(volumeIds), aws.StringValue(volume.FileSystemId))

	volumeIds, err = getAllEfsVolumes(session, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EFS Volumes")
	}

	assert.Contains(t, aws.StringValueSlice(volumeIds), aws.StringValue(volume.FileSystemId))
}

func TestListEFSVolumesWithConfigFile(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	includedEFSVolumeName := "cloud-nuke-test-include-" + util.UniqueID()
	excludedEFSVolumeName := "cloud-nuke-test-" + util.UniqueID()
	az := aws.StringValue(session.Config.Region) + "a"

	includedVolume := createTestEFSVolume(t, session, includedEFSVolumeName, az)
	excludedVolume := createTestEFSVolume(t, session, excludedEFSVolumeName, az)
	defer nukeAllEfsVolumes(session, []*string{includedVolume.FileSystemId, excludedVolume.FileSystemId})

	volumeIds, err := getAllEfsVolumes(session, time.Now().Add(1*time.Hour), config.Config{
		EFSInstances: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{RE: *regexp.MustCompile("^cloud-nuke-test-include-.*")},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(volumeIds))
	require.Equal(t, aws.StringValue(includedVolume.FileSystemId), aws.StringValue(volumeIds[0]))
}

func TestNukeEFSVolume(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	az := aws.StringValue(session.Config.Region) + "a"
	volume := createTestEFSVolume(t, session, uniqueTestID, az)

	volumeIds := findEFSVolumesByNameTag(t, session, uniqueTestID)

	assert.Len(t, volumeIds, 1)
	assert.Equal(t, aws.StringValue(volume.FileSystemId), aws.StringValue(volumeIds[0]))

	if err := nukeAllEfsVolumes(session, volumeIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds, err = getAllEfsVolumes(session, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EFS volumes")
	}

	assert.NotContains(t, aws.StringValueSlice(volumeIds), aws.StringValue(volume.FileSystemId))
}

func TestNukeEFSVolumeWithMountTargets(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	instance := createTestEC2Instance(t, session, uniqueTestID, true)
	az := getAZFromSubnet(t, session, instance.SubnetId)
	volume := createTestEFSVolume(t, session, uniqueTestID, az)
	_ = createTestEFSVolumeMountTarget(t, session, volume.FileSystemId, instance.SubnetId)

	defer nukeAllEfsVolumes(session, []*string{volume.FileSystemId})
	defer nukeAllEc2Instances(session, []*string{instance.InstanceId})

	volumeIds := findEFSVolumesByNameTag(t, session, uniqueTestID)

	assert.Len(t, volumeIds, 1)
	assert.Equal(t, aws.StringValue(volume.FileSystemId), aws.StringValue(volumeIds[0]))

	if err := nukeAllEfsVolumes(session, volumeIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds, err = getAllEfsVolumes(session, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EFS volumes")
	}

	assert.NotContains(t, aws.StringValueSlice(volumeIds), aws.StringValue(volume.FileSystemId))
	if err = removeEC2InstanceProtection(ec2.New(session), &instance); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
}

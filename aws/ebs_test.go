package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/aws-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestEBSVolume(t *testing.T, session *session.Session, name string) ec2.Volume {
	svc := ec2.New(session)
	volume, err := svc.CreateVolume(&ec2.CreateVolumeInput{
		AvailabilityZone: awsgo.String(awsgo.StringValue(session.Config.Region) + "a"),
		Size:             awsgo.Int64(8),
	})

	if err != nil {
		assert.Failf(t, "Could not create test EBS volume: %s", errors.WithStackTrace(err).Error())
	}

	err = svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: []*string{awsgo.String(*volume.VolumeId)},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// Add test tag to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{volume.VolumeId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String("Name"),
				Value: awsgo.String(name),
			},
		},
	})

	if err != nil {
		assert.Failf(t, "Could not tag EBS volume: %s", errors.WithStackTrace(err).Error())
	}

	return *volume
}

func findEBSVolumesByNameTag(output *ec2.DescribeVolumesOutput, name string) []*string {
	var volumeIds []*string
	for _, volume := range output.Volumes {
		// Retrieve only IDs of instances with the unique test tag
		for _, tag := range volume.Tags {
			if awsgo.StringValue(tag.Key) == "Name" && awsgo.StringValue(tag.Value) == name {
				volumeIds = append(volumeIds, volume.VolumeId)
			}
		}
	}

	return volumeIds
}

func TestListEBSVolumes(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "aws-nuke-test-" + util.UniqueID()
	volume := createTestEBSVolume(t, session, uniqueTestID)
	// clean up after this test
	defer nukeAllEbsVolumes(session, []*string{volume.VolumeId})

	volumeIds, err := getAllEbsVolumes(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	assert.NotContains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))

	volumeIds, err = getAllEbsVolumes(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	assert.Contains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))
}

func TestNukeEBSVolumes(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "aws-nuke-test-" + util.UniqueID()
	volume := createTestEBSVolume(t, session, uniqueTestID)

	output, err := ec2.New(session).DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds := findEBSVolumesByNameTag(output, uniqueTestID)

	assert.Len(t, volumeIds, 1)
	assert.Equal(t, awsgo.StringValue(volume.VolumeId), awsgo.StringValue(volumeIds[0]))

	if err := nukeAllEbsVolumes(session, volumeIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds, err = getAllEbsVolumes(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	assert.NotContains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))
}

func TestNukeEBSVolumesInUse(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	svc := ec2.New(session)

	uniqueTestID := "aws-nuke-test-" + util.UniqueID()
	volume := createTestEBSVolume(t, session, uniqueTestID)
	instance := createTestEC2Instance(t, session, uniqueTestID, true)

	defer nukeAllEbsVolumes(session, []*string{volume.VolumeId})
	defer nukeAllEc2Instances(session, []*string{instance.InstanceId})

	// attach volume to protected instance
	svc.AttachVolume(&ec2.AttachVolumeInput{
		Device:     awsgo.String("/dev/sdf"),
		InstanceId: instance.InstanceId,
		VolumeId:   volume.VolumeId,
	})

	output, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds := findEBSVolumesByNameTag(output, uniqueTestID)

	assert.Len(t, volumeIds, 1)
	assert.Equal(t, awsgo.StringValue(volume.VolumeId), awsgo.StringValue(volumeIds[0]))

	if err := nukeAllEbsVolumes(session, volumeIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds, err = getAllEbsVolumes(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	// Volumes should still be in returned slice
	assert.Contains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))
	// remove protection so instance can be cleaned up
	removeEC2InstanceProtection(svc, &instance)
}

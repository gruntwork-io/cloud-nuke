package aws

import (
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getAZFromSubnet(t *testing.T, session *session.Session, subnetID *string) string {
	subnetOutput, err := ec2.New(session).DescribeSubnets(&ec2.DescribeSubnetsInput{
		SubnetIds: []*string{subnetID},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	return *subnetOutput.Subnets[0].AvailabilityZone
}

func createTestEBSVolume(t *testing.T, session *session.Session, name string, az string) ec2.Volume {
	svc := ec2.New(session)
	volume, err := svc.CreateVolume(&ec2.CreateVolumeInput{
		AvailabilityZone: awsgo.String(az),
		Size:             awsgo.Int64(8),
	})

	if err != nil {
		assert.Failf(t, "Could not create test EBS volume", errors.WithStackTrace(err).Error())
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
		assert.Failf(t, "Could not tag EBS volume", errors.WithStackTrace(err).Error())
	}

	return *volume
}

func findEBSVolumesByNameTag(t *testing.T, session *session.Session, name string) []*string {
	output, err := ec2.New(session).DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

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

func findallEBSVolumesByStatus(t *testing.T, session *session.Session, status string) ([]*string, error) {
	statusFilter := ec2.Filter{Name: aws.String("status"), Values: []*string{aws.String(status)}}

	output, err := ec2.New(session).DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{&statusFilter},
	})
	if err != nil {
		return []*string{}, err
	}

	var volumeIds []*string
	for _, volume := range output.Volumes {
		volumeIds = append(volumeIds, volume.VolumeId)
	}

	return volumeIds, nil
}

func TestListEBSVolumes(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
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
	az := awsgo.StringValue(session.Config.Region) + "a"
	volume := createTestEBSVolume(t, session, uniqueTestID, az)
	// clean up after this test
	defer nukeAllEbsVolumes(session, []*string{volume.VolumeId})

	volumeIds, err := getAllEbsVolumes(session, region, time.Now().Add(1*time.Hour*-1), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	assert.NotContains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))

	volumeIds, err = getAllEbsVolumes(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	assert.Contains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))
}

func TestListEBSVolumesWithConfigFile(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	includedEBSVolumeName := "cloud-nuke-test-include-" + util.UniqueID()
	excludedEBSVolumeName := "cloud-nuke-test-" + util.UniqueID()
	az := awsgo.StringValue(session.Config.Region) + "a"

	includedVolume := createTestEBSVolume(t, session, includedEBSVolumeName, az)
	excludedVolume := createTestEBSVolume(t, session, excludedEBSVolumeName, az)
	// clean up after this test
	defer nukeAllEbsVolumes(session, []*string{includedVolume.VolumeId, excludedVolume.VolumeId})

	volumeIds, err := getAllEbsVolumes(session, region, time.Now().Add(1*time.Hour), config.Config{
		EBSVolume: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{RE: *regexp.MustCompile("^cloud-nuke-test-include-.*")},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(volumeIds))
	require.Equal(t, aws.StringValue(includedVolume.VolumeId), aws.StringValue(volumeIds[0]))
}

func TestNukeEBSVolumes(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
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
	az := awsgo.StringValue(session.Config.Region) + "a"
	volume := createTestEBSVolume(t, session, uniqueTestID, az)

	volumeIds := findEBSVolumesByNameTag(t, session, uniqueTestID)

	assert.Len(t, volumeIds, 1)
	assert.Equal(t, awsgo.StringValue(volume.VolumeId), awsgo.StringValue(volumeIds[0]))

	if err := nukeAllEbsVolumes(session, volumeIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds, err = getAllEbsVolumes(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	assert.NotContains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))
}

func TestNukeEBSVolumesInUse(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
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

	svc := ec2.New(session)

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	instance := createTestEC2Instance(t, session, uniqueTestID, true)
	az := getAZFromSubnet(t, session, instance.SubnetId)
	volume := createTestEBSVolume(t, session, uniqueTestID, az)

	defer nukeAllEbsVolumes(session, []*string{volume.VolumeId})
	defer nukeAllEc2Instances(session, []*string{instance.InstanceId})

	// attach volume to protected instance
	_, err = svc.AttachVolume(&ec2.AttachVolumeInput{
		Device:     awsgo.String("/dev/sdf"),
		InstanceId: instance.InstanceId,
		VolumeId:   volume.VolumeId,
	})

	if err != nil {
		assert.Failf(t, "Volume could not be attached", errors.WithStackTrace(err).Error())
	}

	err = svc.WaitUntilVolumeInUse(&ec2.DescribeVolumesInput{
		VolumeIds: []*string{volume.VolumeId},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds := findEBSVolumesByNameTag(t, session, uniqueTestID)

	assert.Len(t, volumeIds, 1)
	assert.Equal(t, awsgo.StringValue(volume.VolumeId), awsgo.StringValue(volumeIds[0]))

	if err := nukeAllEbsVolumes(session, volumeIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds, err = findallEBSVolumesByStatus(t, session, "in-use")
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	// Volumes should still be in returned slice
	assert.Contains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))
	// remove protection so instance can be cleaned up
	if err = removeEC2InstanceProtection(svc, &instance); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
}

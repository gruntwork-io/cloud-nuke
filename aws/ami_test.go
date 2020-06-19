package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func waitUntilImageAvailable(svc *ec2.EC2, input *ec2.DescribeImagesInput) error {
	for i := 0; i < 70; i++ {
		output, err := svc.DescribeImages(input)
		if err != nil {
			return err
		}

		if *output.Images[0].State == "available" {
			return nil
		}

		logging.Logger.Debug("Waiting for ELB to be available")
		time.Sleep(10 * time.Second)
	}

	return ImageAvailableError{}
}

func createTestAMI(t *testing.T, session *session.Session, name string) (*ec2.Image, error) {
	svc := ec2.New(session)
	instance, err := CreateTestEC2Instance(session, name, false)
	if err != nil {
		assert.Failf(t, "Could not create test EC2 instance", errors.WithStackTrace(err).Error())
	}
	output, err := svc.CreateImage(&ec2.CreateImageInput{
		InstanceId: instance.InstanceId,
		Name:       awsgo.String(name),
	})

	if err != nil {
		assert.Failf(t, "Could not create test AMI", errors.WithStackTrace(err).Error())
	}

	params := &ec2.DescribeImagesInput{
		Owners:   []*string{awsgo.String("self")},
		ImageIds: []*string{output.ImageId},
	}

	err = svc.WaitUntilImageExists(params)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	err = svc.WaitUntilImageAvailable(params)

	if err != nil {
		// clean this up since we won't use it again
		defer nukeAllAMIs(session, []*string{output.ImageId})
		return nil, errors.WithStackTrace(err)
	}

	images, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		ImageIds: []*string{output.ImageId},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	return images.Images[0], nil
}

func TestListAMIs(t *testing.T) {
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
	image, err := createTestAMI(t, session, uniqueTestID)
	attempts := 0

	for err != nil && attempts <= 10 {
		// Image didn't become availabe in time, try again
		image, err = createTestAMI(t, session, uniqueTestID)
		attempts++
	}

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// clean up after this test
	defer nukeAllAMIs(session, []*string{image.ImageId})
	instances, err := findEC2InstancesByNameTag(session, uniqueTestID)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	defer nukeAllEc2Instances(session, instances, true)

	amis, err := getAllAMIs(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of AMIs")
	}

	assert.NotContains(t, awsgo.StringValueSlice(amis), *image.ImageId)

	amis, err = getAllAMIs(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of AMIs")
	}

	assert.Contains(t, awsgo.StringValueSlice(amis), *image.ImageId)
}

func TestNukeAMIs(t *testing.T) {
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
	image, err := createTestAMI(t, session, uniqueTestID)
	attempts := 0

	for err != nil && attempts <= 10 {
		// Image didn't become availabe in time, try again
		image, err = createTestAMI(t, session, uniqueTestID)
		attempts++
	}

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// clean up ec2 instance created by the above call
	instances, err := findEC2InstancesByNameTag(session, uniqueTestID)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	defer nukeAllEc2Instances(session, instances, true)

	_, err = svc.DescribeImages(&ec2.DescribeImagesInput{
		ImageIds: []*string{image.ImageId},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if err := nukeAllAMIs(session, []*string{image.ImageId}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	amis, err := getAllAMIs(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of AMIs")
	}

	assert.NotContains(t, awsgo.StringValueSlice(amis), *image.ImageId)
}

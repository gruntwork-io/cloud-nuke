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
	for i := 0; i < 50; i++ {
		output, err := svc.DescribeImages(input)
		if err != nil {
			return err
		}

		if *output.Images[0].State == "available" {
			return nil
		}

		logging.Logger.Debug("Waiting for ELB to be available")
		time.Sleep(5 * time.Second)
	}

	return ImageAvailableError{}
}

func createTestAMI(t *testing.T, session *session.Session, name string) ec2.Image {
	svc := ec2.New(session)
	instance := createTestEC2Instance(t, session, name, false)
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

	err = waitUntilImageAvailable(svc, params)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	images, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		ImageIds: []*string{output.ImageId},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	return *images.Images[0]
}

func TestListAMIs(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	image := createTestAMI(t, session, uniqueTestID)
	// clean up after this test
	defer nukeAllAMIs(session, []*string{image.ImageId})
	defer nukeAllEc2Instances(session, findEC2InstancesByNameTag(t, session, uniqueTestID))

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

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	svc := ec2.New(session)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	image := createTestAMI(t, session, uniqueTestID)

	// clean up ec2 instance created by the above call
	defer nukeAllEc2Instances(session, findEC2InstancesByNameTag(t, session, uniqueTestID))

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

package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of AMI Image ids
func getAllAMIs(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)

	params := &ec2.DescribeImagesInput{
		Owners: []*string{awsgo.String("self")},
	}

	output, err := svc.DescribeImages(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var imageIds []*string
	for _, image := range output.Images {
		layout := "2006-01-02T15:04:05.000Z"
		createdTime, err := time.Parse(layout, *image.CreationDate)
		if err != nil {
			return nil, err
		}

		if excludeAfter.After(createdTime) {
			imageIds = append(imageIds, image.ImageId)
		}
	}

	return imageIds, nil
}

// Deletes all AMIs
func nukeAllAMIs(session *session.Session, imageIds []*string) error {
	svc := ec2.New(session)

	if len(imageIds) == 0 {
		logging.Logger.Infof("No AMIs to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all AMIs in region %s", *session.Config.Region)

	deletedCount := 0
	for _, imageID := range imageIds {
		params := &ec2.DeregisterImageInput{
			ImageId: imageID,
		}

		_, err := svc.DeregisterImage(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedCount++
			logging.Logger.Infof("Deleted AMI: %s", *imageID)
		}
	}

	logging.Logger.Infof("[OK] %d AMI(s) terminated in %s", deletedCount, *session.Config.Region)
	return nil
}

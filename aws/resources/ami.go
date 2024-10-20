package resources

import (
	"context"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/util"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of AMI Image ids
func (ami *AMIs) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var imageIds []*string
	paginator := ec2.NewDescribeImagesPaginator(ami.Client, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, image := range page.Images {

			createdTime, errTimeParse := util.ParseTimestamp(image.CreationDate)
			if errTimeParse != nil {
				return nil, errTimeParse
			}

			// Check if the image has a tag that indicates AWS management
			isAWSManaged := false
			for _, tag := range image.Tags {
				if *tag.Key == "aws-managed" && *tag.Value == "true" {
					isAWSManaged = true
					break
				}
			}

			// Skip AWS managed images and images created by AWS Backup
			if isAWSManaged || strings.HasPrefix(*image.Name, "AwsBackup") {
				continue
			}

			if configObj.AMI.ShouldInclude(config.ResourceValue{
				Name: image.Name,
				Time: createdTime,
			}) {
				imageIds = append(imageIds, image.ImageId)
			}

		}
	}

	// checking the nukable permissions
	ami.VerifyNukablePermissions(imageIds, func(id *string) error {
		_, err := ami.Client.DeregisterImage(ami.Context, &ec2.DeregisterImageInput{
			ImageId: id,
			DryRun:  aws.Bool(true),
		})
		return err
	})

	return imageIds, nil
}

// Deletes all AMI
func (ami *AMIs) nukeAll(imageIds []*string) error {
	if len(imageIds) == 0 {
		logging.Debugf("No AMI to nuke in region %s", ami.Region)
		return nil

	}

	logging.Debugf("Deleting all AMI in region %s", ami.Region)

	deletedCount := 0
	for _, imageID := range imageIds {
		if nukable, reason := ami.IsNukable(aws.ToString(imageID)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(imageID), reason)
			continue
		}

		_, err := ami.Client.DeregisterImage(ami.Context, &ec2.DeregisterImageInput{
			ImageId: imageID,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(imageID),
			ResourceType: "Amazon Machine Image (AMI)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedCount++
			logging.Debugf("Deleted AMI: %s", *imageID)
		}
	}

	logging.Debugf("[OK] %d AMI(s) terminated in %s", deletedCount, ami.Region)
	return nil
}

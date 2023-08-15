package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of AMI Image ids
func (ami *AMIs) getAll(configObj config.Config) ([]*string, error) {
	params := &ec2.DescribeImagesInput{
		Owners: []*string{awsgo.String("self")},
	}

	output, err := ami.Client.DescribeImages(params)
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

		if configObj.AMI.ShouldInclude(config.ResourceValue{
			Name: image.Name,
			Time: &createdTime,
		}) {
			imageIds = append(imageIds, image.ImageId)
		}
	}

	return imageIds, nil
}

// Deletes all AMI
func (ami *AMIs) nukeAll(imageIds []*string) error {
	if len(imageIds) == 0 {
		logging.Logger.Debugf("No AMI to nuke in region %s", ami.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all AMI in region %s", ami.Region)

	deletedCount := 0
	for _, imageID := range imageIds {
		params := &ec2.DeregisterImageInput{
			ImageId: imageID,
		}

		_, err := ami.Client.DeregisterImage(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(imageID),
			ResourceType: "Amazon Machine Image (AMI)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking AMI",
			}, map[string]interface{}{
				"region": ami.Region,
			})
		} else {
			deletedCount++
			logging.Logger.Debugf("Deleted AMI: %s", *imageID)
		}
	}

	logging.Logger.Debugf("[OK] %d AMI(s) terminated in %s", deletedCount, ami.Region)
	return nil
}

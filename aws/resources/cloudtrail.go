package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (ct *CloudtrailTrail) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	param := &cloudtrail.ListTrailsInput{}

	trailIds := []*string{}
	paginator := func(output *cloudtrail.ListTrailsOutput, lastPage bool) bool {
		for _, trailInfo := range output.Trails {
			if configObj.CloudtrailTrail.ShouldInclude(config.ResourceValue{
				Name: trailInfo.Name,
			}) {
				trailIds = append(trailIds, trailInfo.TrailARN)
			}
		}

		return !lastPage
	}

	err := ct.Client.ListTrailsPages(param, paginator)
	if err != nil {
		return trailIds, errors.WithStackTrace(err)
	}

	return trailIds, nil
}

func (ct *CloudtrailTrail) nukeAll(arns []*string) error {
	if len(arns) == 0 {
		logging.Debugf("No Cloudtrail Trails to nuke in region %s", ct.Region)
		return nil
	}

	logging.Debugf("Deleting all Cloudtrail Trails in region %s", ct.Region)
	var deletedArns []*string

	for _, arn := range arns {
		params := &cloudtrail.DeleteTrailInput{
			Name: arn,
		}

		_, err := ct.Client.DeleteTrail(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(arn),
			ResourceType: "Cloudtrail Trail",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Cloudtrail",
			}, map[string]interface{}{
				"region": ct.Region,
			})
		} else {
			deletedArns = append(deletedArns, arn)
			logging.Debugf("Deleted Cloudtrail Trail: %s", aws.StringValue(arn))
		}
	}

	logging.Debugf("[OK] %d Cloudtrail Trail deleted in %s", len(deletedArns), ct.Region)

	return nil
}

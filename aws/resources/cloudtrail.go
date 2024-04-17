package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
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
		} else {
			deletedArns = append(deletedArns, arn)
			logging.Debugf("Deleted Cloudtrail Trail: %s", aws.StringValue(arn))
		}
	}

	logging.Debugf("[OK] %d Cloudtrail Trail deleted in %s", len(deletedArns), ct.Region)

	return nil
}

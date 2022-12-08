package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllCloudtrailTrails(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := cloudtrail.New(session)

	param := &cloudtrail.ListTrailsInput{}

	trailIds := []*string{}

	paginator := func(output *cloudtrail.ListTrailsOutput, lastPage bool) bool {
		for _, trailInfo := range output.Trails {
			if shouldIncludeCloudtrailTrail(trailInfo, configObj) {
				trailIds = append(trailIds, trailInfo.TrailARN)
			}
		}
		return !lastPage
	}

	err := svc.ListTrailsPages(param, paginator)
	if err != nil {
		return trailIds, errors.WithStackTrace(err)
	}

	return trailIds, nil
}

func shouldIncludeCloudtrailTrail(trail *cloudtrail.TrailInfo, configObj config.Config) bool {
	if trail == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(trail.Name),
		configObj.CloudtrailTrail.IncludeRule.NamesRegExp,
		configObj.CloudtrailTrail.ExcludeRule.NamesRegExp,
	)
}

func nukeAllCloudTrailTrails(session *session.Session, arns []*string) error {
	svc := cloudtrail.New(session)

	if len(arns) == 0 {
		logging.Logger.Debugf("No Cloudtrail Trails to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Cloudtrail Trails in region %s", *session.Config.Region)
	var deletedArns []*string

	for _, arn := range arns {
		params := &cloudtrail.DeleteTrailInput{
			Name: arn,
		}

		_, err := svc.DeleteTrail(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(arn),
			ResourceType: "Cloudtrail Trail",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
		} else {
			deletedArns = append(deletedArns, arn)
			logging.Logger.Debugf("Deleted Cloudtrail Trail: %s", aws.StringValue(arn))
		}
	}

	logging.Logger.Debugf("[OK] %d Cloudtrail Trail deleted in %s", len(deletedArns), *session.Config.Region)

	return nil
}

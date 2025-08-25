package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (ct *CloudtrailTrail) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var trailIds []*string
	paginator := cloudtrail.NewListTrailsPaginator(ct.Client, &cloudtrail.ListTrailsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Process trails individually to avoid CloudTrailARNInvalidException when mixing
		// organization trails (from AWS Control Tower) with account-level trails.
		// AWS ListTags API doesn't allow resources from multiple owners in a single call.
		for _, trail := range page.Trails {
			rv := config.ResourceValue{Name: trail.Name, Tags: make(map[string]string)}

			if tags, err := ct.Client.ListTags(c, &cloudtrail.ListTagsInput{
				ResourceIdList: []string{*trail.TrailARN},
			}); err == nil && len(tags.ResourceTagList) > 0 {
				for _, tag := range tags.ResourceTagList[0].TagsList {
					rv.Tags[*tag.Key] = *tag.Value
				}
			}

			if configObj.CloudtrailTrail.ShouldInclude(rv) {
				trailIds = append(trailIds, trail.TrailARN)
			}
		}
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

		_, err := ct.Client.DeleteTrail(ct.Context, params)
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(arn),
			ResourceType: "Cloudtrail Trail",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedArns = append(deletedArns, arn)
			logging.Debugf("Deleted Cloudtrail Trail: %s", aws.ToString(arn))
		}
	}

	logging.Debugf("[OK] %d Cloudtrail Trail deleted in %s", len(deletedArns), ct.Region)

	return nil
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func (r *Route53TrafficPolicy) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var ids []*string

	result, err := r.Client.ListTrafficPolicies(&route53.ListTrafficPoliciesInput{})
	if err != nil {
		logging.Errorf("[Failed] unable to list traffic policies: %v", err)
		return nil, err
	}

	for _, summary := range result.TrafficPolicySummaries {
		if configObj.Route53TrafficPolicy.ShouldInclude(config.ResourceValue{
			Name: summary.Name,
		}) {
			ids = append(ids, summary.Id)
			// store the corresponding version, as this is required for nuking
			r.versionMap[*summary.Id] = summary.LatestVersion
		}
	}
	return ids, nil
}

func (r *Route53TrafficPolicy) nukeAll(identifiers []*string) (err error) {
	if len(identifiers) == 0 {
		logging.Debugf("No Route53 Traffic Policy to nuke in region %s", r.Region)
		return nil
	}
	logging.Debugf("Deleting all Route53 Traffic Policy in region %s", r.Region)

	var deletedIds []*string
	for _, id := range identifiers {
		_, err := r.Client.DeleteTrafficPolicy(&route53.DeleteTrafficPolicyInput{
			Id:      id,
			Version: r.versionMap[*id],
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "Route53 Traffic Policy",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *id, err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Route53 Traffic Policy: %s", aws.StringValue(id))
		}
	}

	logging.Debugf("[OK] %d Route53 Traffic Policy(s) deleted in %s", len(deletedIds), r.Region)

	return nil
}

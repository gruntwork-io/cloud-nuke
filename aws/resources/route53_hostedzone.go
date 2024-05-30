package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func (r *Route53HostedZone) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var ids []*string

	result, err := r.Client.ListHostedZonesWithContext(r.Context, &route53.ListHostedZonesInput{})
	if err != nil {
		logging.Errorf("[Failed] unable to list hosted-zones: %s", err)
		return nil, err
	}

	for _, r := range result.HostedZones {
		if configObj.Route53HostedZone.ShouldInclude(config.ResourceValue{
			Name: r.Name,
		}) {
			ids = append(ids, r.Id)
		}
	}
	return ids, nil
}

func (r *Route53HostedZone) nukeAll(identifiers []*string) (err error) {
	if len(identifiers) == 0 {
		logging.Debugf("No Route53 Hosted Zones to nuke in region %s", r.Region)
		return nil
	}
	logging.Debugf("Deleting all Route53 Hosted Zones in region %s", r.Region)

	var deletedIds []*string
	for _, id := range identifiers {
		_, err := r.Client.DeleteHostedZoneWithContext(r.Context, &route53.DeleteHostedZoneInput{
			Id: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "Route53 hosted zone",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *id, err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Route53 Hosted Zone: %s", aws.StringValue(id))
		}
	}

	logging.Debugf("[OK] %d Route53 hosted zone(s) deleted in %s", len(deletedIds), r.Region)

	return nil
}

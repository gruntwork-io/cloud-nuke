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

// IMPORTANT:
// (https://docs.aws.amazon.com/Route53/latest/APIReference/API_DeleteTrafficPolicyInstance.html).
// Amazon Route 53 will delete the resource record set automatically. If you delete the resource record set by using ChangeResourceRecordSets,
// Route 53 doesn't automatically delete the traffic policy instance, and you'll continue to be charged for it even though it's no longer in use.
func (r *Route53HostedZone) nukeTrafficPolicy(id *string) (err error) {
	logging.Debugf("[Traffic Policy] nuking the traffic policy attached with the hosted zone")

	_, err = r.Client.DeleteTrafficPolicyInstance(&route53.DeleteTrafficPolicyInstanceInput{
		Id: id,
	})
	return err
}

func (r *Route53HostedZone) nukeHostedZone(id *string) (err error) {

	_, err = r.Client.DeleteHostedZoneWithContext(r.Context, &route53.DeleteHostedZoneInput{
		Id: id,
	})

	return err
}

func (r *Route53HostedZone) nukeRecordSet(id *string) (err error) {

	// get the resource records
	output, err := r.Client.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId: id,
	})
	if err != nil {
		logging.Errorf("[Failed] unable to list resource record set: %s", err)
		return err
	}

	var changes []*route53.Change
	for _, record := range output.ResourceRecordSets {
		if aws.StringValue(record.Type) == "NS" || aws.StringValue(record.Type) == "SOA" {
			logging.Infof("[Skipping] resource record set type is : %s", aws.StringValue(record.Type))
			continue
		}

		// Note : the request shoud contain exactly one of [AliasTarget, all of [TTL, and ResourceRecords], or TrafficPolicyInstanceId]
		if record.TrafficPolicyInstanceId != nil {
			// nuke the traffic policy
			err := r.nukeTrafficPolicy(record.TrafficPolicyInstanceId)
			if err != nil {
				logging.Errorf("[Failed] unable to nuke traffic policy: %s", err)
				return err
			}

			record.ResourceRecords = nil
		}

		// set the changes slice
		changes = append(changes, &route53.Change{
			Action:            aws.String("DELETE"),
			ResourceRecordSet: record,
		})
	}

	if len(changes) > 0 {
		_, err = r.Client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: id,
			ChangeBatch: &route53.ChangeBatch{
				Changes: changes,
			},
		})

		if err != nil {
			logging.Errorf("[Failed] unable to nuke resource record set: %s", err)
			return err
		}
	}

	return nil
}

func (r *Route53HostedZone) nuke(id *string) (err error) {

	err = r.nukeRecordSet(id)
	if err != nil {
		logging.Errorf("[Failed] unable to nuke record sets: %s", err)
		return err
	}

	err = r.nukeHostedZone(id)
	if err != nil {
		logging.Errorf("[Failed] unable to nuke hosted zone: %s", err)
		return err
	}

	return nil
}

func (r *Route53HostedZone) nukeAll(identifiers []*string) (err error) {
	if len(identifiers) == 0 {
		logging.Debugf("No Route53 Hosted Zones to nuke in region %s", r.Region)
		return nil
	}
	logging.Debugf("Deleting all Route53 Hosted Zones in region %s", r.Region)

	var deletedIds []*string
	for _, id := range identifiers {

		err = r.nuke(id)
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

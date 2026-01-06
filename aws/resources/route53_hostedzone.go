package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// Route53HostedZoneAPI defines the interface for Route53 hosted zone operations.
type Route53HostedZoneAPI interface {
	ListHostedZones(ctx context.Context, params *route53.ListHostedZonesInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error)
	ListTagsForResources(ctx context.Context, params *route53.ListTagsForResourcesInput, optFns ...func(*route53.Options)) (*route53.ListTagsForResourcesOutput, error)
	ListResourceRecordSets(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
	ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
	DeleteHostedZone(ctx context.Context, params *route53.DeleteHostedZoneInput, optFns ...func(*route53.Options)) (*route53.DeleteHostedZoneOutput, error)
	DeleteTrafficPolicyInstance(ctx context.Context, params *route53.DeleteTrafficPolicyInstanceInput, optFns ...func(*route53.Options)) (*route53.DeleteTrafficPolicyInstanceOutput, error)
}

// NewRoute53HostedZone creates a new Route53 Hosted Zone resource using the generic resource pattern.
func NewRoute53HostedZone() AwsResource {
	return NewAwsResource(&resource.Resource[Route53HostedZoneAPI]{
		ResourceTypeName: "route53-hosted-zone",
		BatchSize:        DefaultBatchSize,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[Route53HostedZoneAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = route53.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.Route53HostedZone
		},
		Lister: listRoute53HostedZones,
		Nuker:  resource.SequentialDeleter(deleteRoute53HostedZone),
	})
}

// listRoute53HostedZones retrieves all Route53 hosted zones that match the config filters.
// Returns identifiers in format "zoneId|domainName" since deletion needs the domain name
// to identify which NS/SOA records to skip.
func listRoute53HostedZones(ctx context.Context, client Route53HostedZoneAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := route53.NewListHostedZonesPaginator(client, &route53.ListHostedZonesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		// Collect zone IDs for batch tag lookup
		zoneIds := make([]string, 0, len(page.HostedZones))
		for _, zone := range page.HostedZones {
			zoneIds = append(zoneIds, strings.TrimPrefix(aws.ToString(zone.Id), "/hostedzone/"))
		}

		// Fetch tags for all zones in this page
		tagsByZoneId := make(map[string][]types.Tag)
		if len(zoneIds) > 0 {
			tagsOutput, err := client.ListTagsForResources(ctx, &route53.ListTagsForResourcesInput{
				ResourceType: types.TagResourceTypeHostedzone,
				ResourceIds:  zoneIds,
			})
			if err != nil {
				return nil, err
			}
			for _, tagSet := range tagsOutput.ResourceTagSets {
				tagsByZoneId[aws.ToString(tagSet.ResourceId)] = tagSet.Tags
			}
		}

		// Filter zones based on config
		for _, zone := range page.HostedZones {
			zoneId := strings.TrimPrefix(aws.ToString(zone.Id), "/hostedzone/")
			tags := util.ConvertRoute53TagsToMap(tagsByZoneId[zoneId])

			if cfg.ShouldInclude(config.ResourceValue{
				Name: zone.Name,
				Tags: tags,
			}) {
				// Encode both zone ID and domain name in the identifier
				identifier := fmt.Sprintf("%s|%s", aws.ToString(zone.Id), aws.ToString(zone.Name))
				identifiers = append(identifiers, aws.String(identifier))
			}
		}
	}

	return identifiers, nil
}

// deleteRoute53HostedZone deletes a single Route53 hosted zone and all its record sets.
// Expects identifier in format "zoneId|domainName".
func deleteRoute53HostedZone(ctx context.Context, client Route53HostedZoneAPI, identifier *string) error {
	zoneId, domainName, err := parseHostedZoneIdentifier(aws.ToString(identifier))
	if err != nil {
		return err
	}

	// Step 1: Delete all record sets (except required SOA and NS records)
	if err := deleteHostedZoneRecordSets(ctx, client, zoneId, domainName); err != nil {
		return err
	}

	// Step 2: Delete the hosted zone
	_, err = client.DeleteHostedZone(ctx, &route53.DeleteHostedZoneInput{
		Id: aws.String(zoneId),
	})
	return err
}

// parseHostedZoneIdentifier parses an identifier in format "zoneId|domainName".
func parseHostedZoneIdentifier(identifier string) (string, string, error) {
	parts := strings.SplitN(identifier, "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid hosted zone identifier format: %s", identifier)
	}
	return parts[0], parts[1], nil
}

// deleteHostedZoneRecordSets deletes all deletable record sets from a hosted zone.
func deleteHostedZoneRecordSets(ctx context.Context, client Route53HostedZoneAPI, zoneId, domainName string) error {
	var changes []types.Change

	paginator := route53.NewListResourceRecordSetsPaginator(client, &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneId),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, record := range page.ResourceRecordSets {
			// Skip required SOA and NS records at the zone apex
			// Reference: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resource-record-sets-deleting.html
			if (record.Type == types.RRTypeNs || record.Type == types.RRTypeSoa) && aws.ToString(record.Name) == domainName {
				logging.Debugf("Skipping required %s record for zone apex", record.Type)
				continue
			}

			// Handle traffic policy instances
			// Reference: https://docs.aws.amazon.com/Route53/latest/APIReference/API_DeleteTrafficPolicyInstance.html
			if record.TrafficPolicyInstanceId != nil {
				if err := deleteTrafficPolicyInstance(ctx, client, record.TrafficPolicyInstanceId); err != nil {
					return err
				}
				record.ResourceRecords = nil
			}

			changes = append(changes, types.Change{
				Action:            types.ChangeActionDelete,
				ResourceRecordSet: &record,
			})
		}
	}

	if len(changes) > 0 {
		_, err := client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: aws.String(zoneId),
			ChangeBatch: &types.ChangeBatch{
				Changes: changes,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteTrafficPolicyInstance deletes a traffic policy instance.
// This must be done before deleting the associated record set, otherwise the traffic policy
// instance continues to be charged even though it's no longer in use.
func deleteTrafficPolicyInstance(ctx context.Context, client Route53HostedZoneAPI, instanceId *string) error {
	_, err := client.DeleteTrafficPolicyInstance(ctx, &route53.DeleteTrafficPolicyInstanceInput{
		Id: instanceId,
	})
	return err
}

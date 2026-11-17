package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// NetworkFirewallAPI defines the interface for Network Firewall operations.
type NetworkFirewallAPI interface {
	ListFirewalls(ctx context.Context, params *networkfirewall.ListFirewallsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallsOutput, error)
	DescribeFirewall(ctx context.Context, params *networkfirewall.DescribeFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallOutput, error)
	DeleteFirewall(ctx context.Context, params *networkfirewall.DeleteFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallOutput, error)
}

// NewNetworkFirewalls creates a new Network Firewall resource using the generic resource pattern.
func NewNetworkFirewalls() AwsResource {
	return NewAwsResource(&resource.Resource[NetworkFirewallAPI]{
		ResourceTypeName: "network-firewall",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[NetworkFirewallAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = networkfirewall.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.NetworkFirewall
		},
		Lister:             listNetworkFirewalls,
		Nuker:              resource.SimpleBatchDeleter(deleteNetworkFirewall),
		PermissionVerifier: verifyNetworkFirewallPermissions,
	})
}

// listNetworkFirewalls retrieves all Network Firewalls that match the config filters.
func listNetworkFirewalls(ctx context.Context, client NetworkFirewallAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := networkfirewall.NewListFirewallsPaginator(client, &networkfirewall.ListFirewallsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, firewall := range page.Firewalls {
			output, err := client.DescribeFirewall(ctx, &networkfirewall.DescribeFirewallInput{
				FirewallArn: firewall.FirewallArn,
			})
			if err != nil {
				logging.Errorf("[Failed] to describe firewall %s: %v", aws.ToString(firewall.FirewallArn), err)
				continue
			}

			if output.Firewall == nil {
				logging.Errorf("[Failed] no firewall information found for %s", aws.ToString(firewall.FirewallArn))
				continue
			}

			tags := util.ConvertNetworkFirewallTagsToMap(output.Firewall.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, firewall.FirewallArn, tags)
			if err != nil {
				logging.Errorf("[Failed] to get first seen time for %s: %v", aws.ToString(firewall.FirewallArn), err)
				continue
			}

			if shouldIncludeNetworkFirewall(output.Firewall, firstSeenTime, cfg) {
				identifiers = append(identifiers, firewall.FirewallName)
			}
		}
	}

	return identifiers, nil
}

// shouldIncludeNetworkFirewall determines if a firewall should be included based on config filters.
func shouldIncludeNetworkFirewall(firewall *types.Firewall, firstSeenTime *time.Time, cfg config.ResourceType) bool {
	tags := util.ConvertNetworkFirewallTagsToMap(firewall.Tags)
	identifierName := aws.ToString(firewall.FirewallName)
	if v, ok := tags["Name"]; ok {
		identifierName = v
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: &identifierName,
		Tags: tags,
		Time: firstSeenTime,
	})
}

// verifyNetworkFirewallPermissions checks if the firewall has delete protection enabled.
func verifyNetworkFirewallPermissions(ctx context.Context, client NetworkFirewallAPI, id *string) error {
	output, err := client.DescribeFirewall(ctx, &networkfirewall.DescribeFirewallInput{
		FirewallName: id,
	})
	if err != nil {
		return err
	}

	if output.Firewall != nil && output.Firewall.DeleteProtection {
		return util.ErrDeleteProtectionEnabled
	}
	return nil
}

// deleteNetworkFirewall deletes a single Network Firewall.
func deleteNetworkFirewall(ctx context.Context, client NetworkFirewallAPI, id *string) error {
	_, err := client.DeleteFirewall(ctx, &networkfirewall.DeleteFirewallInput{
		FirewallName: id,
	})
	return err
}

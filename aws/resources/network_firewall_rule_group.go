package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// NetworkFirewallRuleGroupAPI defines the interface for NetworkFirewall Rule Group operations.
type NetworkFirewallRuleGroupAPI interface {
	ListRuleGroups(ctx context.Context, params *networkfirewall.ListRuleGroupsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListRuleGroupsOutput, error)
	DescribeRuleGroup(ctx context.Context, params *networkfirewall.DescribeRuleGroupInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeRuleGroupOutput, error)
	DeleteRuleGroup(ctx context.Context, params *networkfirewall.DeleteRuleGroupInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteRuleGroupOutput, error)
	TagResource(ctx context.Context, params *networkfirewall.TagResourceInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.TagResourceOutput, error)
}

// NewNetworkFirewallRuleGroup creates a new NetworkFirewall Rule Group resource.
func NewNetworkFirewallRuleGroup() AwsResource {
	return NewAwsResource(&resource.Resource[NetworkFirewallRuleGroupAPI]{
		ResourceTypeName: "network-firewall-rule-group",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[NetworkFirewallRuleGroupAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = networkfirewall.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.NetworkFirewallRuleGroup
		},
		Lister: listNetworkFirewallRuleGroups,
		Nuker:  resource.SimpleBatchDeleter(deleteNetworkFirewallRuleGroup),
	})
}

// listNetworkFirewallRuleGroups retrieves all Network Firewall rule groups that match the config filters.
func listNetworkFirewallRuleGroups(ctx context.Context, client NetworkFirewallRuleGroupAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := networkfirewall.NewListRuleGroupsPaginator(client, &networkfirewall.ListRuleGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, group := range page.RuleGroups {
			output, err := client.DescribeRuleGroup(ctx, &networkfirewall.DescribeRuleGroupInput{
				RuleGroupArn: group.Arn,
			})
			if err != nil {
				logging.Errorf("[Failed] to describe firewall rule group %s: %s", aws.ToString(group.Name), err)
				continue
			}

			if output.RuleGroupResponse == nil {
				logging.Debugf("[Skip] no rule group response found for %s", aws.ToString(group.Name))
				continue
			}

			resp := output.RuleGroupResponse

			// Skip rule groups that are still in use
			if aws.ToInt32(resp.NumberOfAssociations) > 0 {
				logging.Debugf("[Skip] rule group %s is still in use", aws.ToString(resp.RuleGroupName))
				continue
			}

			tags := util.ConvertNetworkFirewallTagsToMap(resp.Tags)
			identifierName := aws.ToString(resp.RuleGroupName)
			if nameTag, ok := tags["Name"]; ok {
				identifierName = nameTag
			}

			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, group.Arn, tags)
			if err != nil {
				logging.Errorf("[Failed] to get first seen time for %s: %s", aws.ToString(group.Name), err)
				continue
			}

			if !cfg.ShouldInclude(config.ResourceValue{
				Name: &identifierName,
				Tags: tags,
				Time: firstSeenTime,
			}) {
				continue
			}

			// Use ARN as identifier - DeleteRuleGroup can use ARN without requiring Type
			identifiers = append(identifiers, group.Arn)
		}
	}

	return identifiers, nil
}

// deleteNetworkFirewallRuleGroup deletes a single Network Firewall rule group by ARN.
func deleteNetworkFirewallRuleGroup(ctx context.Context, client NetworkFirewallRuleGroupAPI, arn *string) error {
	_, err := client.DeleteRuleGroup(ctx, &networkfirewall.DeleteRuleGroupInput{
		RuleGroupArn: arn,
	})
	return err
}

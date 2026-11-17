package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// NetworkFirewallPolicyAPI defines the interface for Network Firewall Policy operations.
type NetworkFirewallPolicyAPI interface {
	ListFirewallPolicies(ctx context.Context, params *networkfirewall.ListFirewallPoliciesInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallPoliciesOutput, error)
	DescribeFirewallPolicy(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallPolicyOutput, error)
	DeleteFirewallPolicy(ctx context.Context, params *networkfirewall.DeleteFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallPolicyOutput, error)
	TagResource(ctx context.Context, params *networkfirewall.TagResourceInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.TagResourceOutput, error)
}

// NewNetworkFirewallPolicy creates a new NetworkFirewallPolicy resource using the generic resource pattern.
func NewNetworkFirewallPolicy() AwsResource {
	return NewAwsResource(&resource.Resource[NetworkFirewallPolicyAPI]{
		ResourceTypeName: "network-firewall-policy",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[NetworkFirewallPolicyAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = networkfirewall.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.NetworkFirewallPolicy
		},
		Lister: listNetworkFirewallPolicies,
		Nuker:  resource.SimpleBatchDeleter(deleteNetworkFirewallPolicy),
	})
}

// listNetworkFirewallPolicies retrieves all Network Firewall Policies that match the config filters.
func listNetworkFirewallPolicies(ctx context.Context, client NetworkFirewallPolicyAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := networkfirewall.NewListFirewallPoliciesPaginator(client, &networkfirewall.ListFirewallPoliciesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, policy := range page.FirewallPolicies {
			output, err := client.DescribeFirewallPolicy(ctx, &networkfirewall.DescribeFirewallPolicyInput{
				FirewallPolicyArn: policy.Arn,
			})
			if err != nil {
				logging.Errorf("[Failed] to describe firewall policy %s: %v", aws.ToString(policy.Name), err)
				continue
			}

			if output.FirewallPolicyResponse == nil {
				logging.Errorf("[Failed] no firewall policy information found for %s", aws.ToString(policy.Name))
				continue
			}

			if shouldIncludeNetworkFirewallPolicy(ctx, client, policy.Arn, output.FirewallPolicyResponse, cfg) {
				identifiers = append(identifiers, policy.Name)
			}
		}
	}

	return identifiers, nil
}

// shouldIncludeNetworkFirewallPolicy determines if a firewall policy should be included for deletion.
func shouldIncludeNetworkFirewallPolicy(ctx context.Context, client NetworkFirewallPolicyAPI, arn *string, policy *types.FirewallPolicyResponse, cfg config.ResourceType) bool {
	// Skip policies that are still in use
	if aws.ToInt32(policy.NumberOfAssociations) > 0 {
		logging.Debugf("[Skipping] policy %s is still in use", aws.ToString(policy.FirewallPolicyName))
		return false
	}

	tags := util.ConvertNetworkFirewallTagsToMap(policy.Tags)

	firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, arn, tags)
	if err != nil {
		logging.Errorf("Unable to retrieve first seen time for policy %s: %v", aws.ToString(policy.FirewallPolicyName), err)
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: getNetworkFirewallPolicyName(tags),
		Tags: tags,
		Time: firstSeenTime,
	})
}

// getNetworkFirewallPolicyName returns the Name tag value if present, otherwise nil.
func getNetworkFirewallPolicyName(tags map[string]string) *string {
	if name, ok := tags["Name"]; ok {
		return &name
	}
	return nil
}

// deleteNetworkFirewallPolicy deletes a single Network Firewall Policy.
func deleteNetworkFirewallPolicy(ctx context.Context, client NetworkFirewallPolicyAPI, id *string) error {
	_, err := client.DeleteFirewallPolicy(ctx, &networkfirewall.DeleteFirewallPolicyInput{
		FirewallPolicyName: id,
	})
	return err
}

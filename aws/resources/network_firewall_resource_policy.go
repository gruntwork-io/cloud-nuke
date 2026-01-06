package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// NetworkFirewallResourcePolicyAPI defines the interface for Network Firewall resource policy operations.
// Resource policies are attached to shared firewall policies or rule groups for cross-account sharing.
// References:
// - https://docs.aws.amazon.com/network-firewall/latest/developerguide/security_iam_resource-based-policy-examples.html
// - https://docs.aws.amazon.com/network-firewall/latest/developerguide/sharing.html
type NetworkFirewallResourcePolicyAPI interface {
	ListFirewallPolicies(ctx context.Context, params *networkfirewall.ListFirewallPoliciesInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallPoliciesOutput, error)
	ListRuleGroups(ctx context.Context, params *networkfirewall.ListRuleGroupsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListRuleGroupsOutput, error)
	DescribeResourcePolicy(ctx context.Context, params *networkfirewall.DescribeResourcePolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeResourcePolicyOutput, error)
	DeleteResourcePolicy(ctx context.Context, params *networkfirewall.DeleteResourcePolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteResourcePolicyOutput, error)
}

// NewNetworkFirewallResourcePolicy creates a new Network Firewall resource policy resource.
func NewNetworkFirewallResourcePolicy() AwsResource {
	return NewAwsResource(&resource.Resource[NetworkFirewallResourcePolicyAPI]{
		ResourceTypeName: "network-firewall-resource-policy",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[NetworkFirewallResourcePolicyAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = networkfirewall.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.NetworkFirewallResourcePolicy
		},
		Lister: listNetworkFirewallResourcePolicies,
		Nuker:  resource.SimpleBatchDeleter(deleteNetworkFirewallResourcePolicy),
	})
}

// listNetworkFirewallResourcePolicies retrieves all resource policy ARNs by finding firewall policies
// and rule groups that have resource policies attached.
func listNetworkFirewallResourcePolicies(ctx context.Context, client NetworkFirewallResourcePolicyAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// Collect all resource ARNs (firewall policies + rule groups)
	resourceArns, err := collectResourceArns(ctx, client)
	if err != nil {
		return nil, err
	}

	// Filter to only those with resource policies attached
	var ids []*string
	for _, arn := range resourceArns {
		output, err := client.DescribeResourcePolicy(ctx, &networkfirewall.DescribeResourcePolicyInput{
			ResourceArn: arn,
		})
		if err != nil && util.TransformAWSError(err) != util.ErrResourceNotFoundException {
			return nil, errors.WithStackTrace(err)
		}

		// Include if a policy exists and passes the filter
		if output != nil && output.Policy != nil {
			if cfg.ShouldInclude(config.ResourceValue{Name: arn}) {
				ids = append(ids, arn)
			}
		}
	}

	return ids, nil
}

// collectResourceArns gathers all firewall policy and rule group ARNs with pagination.
func collectResourceArns(ctx context.Context, client NetworkFirewallResourcePolicyAPI) ([]*string, error) {
	var arns []*string

	// List firewall policies with pagination
	policyPaginator := networkfirewall.NewListFirewallPoliciesPaginator(client, &networkfirewall.ListFirewallPoliciesInput{})
	for policyPaginator.HasMorePages() {
		page, err := policyPaginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, policy := range page.FirewallPolicies {
			arns = append(arns, policy.Arn)
		}
	}

	// List rule groups with pagination
	groupPaginator := networkfirewall.NewListRuleGroupsPaginator(client, &networkfirewall.ListRuleGroupsInput{})
	for groupPaginator.HasMorePages() {
		page, err := groupPaginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, group := range page.RuleGroups {
			arns = append(arns, group.Arn)
		}
	}

	return arns, nil
}

// deleteNetworkFirewallResourcePolicy deletes a single resource policy.
func deleteNetworkFirewallResourcePolicy(ctx context.Context, client NetworkFirewallResourcePolicyAPI, arn *string) error {
	_, err := client.DeleteResourcePolicy(ctx, &networkfirewall.DeleteResourcePolicyInput{
		ResourceArn: arn,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

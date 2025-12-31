package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type NetworkFirewallResourcePolicyAPI interface {
	ListFirewallPolicies(ctx context.Context, params *networkfirewall.ListFirewallPoliciesInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallPoliciesOutput, error)
	ListRuleGroups(ctx context.Context, params *networkfirewall.ListRuleGroupsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListRuleGroupsOutput, error)
	DescribeResourcePolicy(ctx context.Context, params *networkfirewall.DescribeResourcePolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeResourcePolicyOutput, error)
	DeleteResourcePolicy(ctx context.Context, params *networkfirewall.DeleteResourcePolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteResourcePolicyOutput, error)
}

type NetworkFirewallResourcePolicy struct {
	BaseAwsResource
	Client      NetworkFirewallResourcePolicyAPI
	Region      string
	Identifiers []string
}

func (nfrp *NetworkFirewallResourcePolicy) Init(cfg aws.Config) {
	nfrp.Client = networkfirewall.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (nfrp *NetworkFirewallResourcePolicy) ResourceName() string {
	return "network-firewall-resource-policy"
}

func (nfrp *NetworkFirewallResourcePolicy) ResourceIdentifiers() []string {
	return nfrp.Identifiers
}

func (nfrp *NetworkFirewallResourcePolicy) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (nfrp *NetworkFirewallResourcePolicy) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkFirewallResourcePolicy
}

func (nfrp *NetworkFirewallResourcePolicy) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nfrp.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nfrp.Identifiers = aws.ToStringSlice(identifiers)
	return nfrp.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (nfrp *NetworkFirewallResourcePolicy) Nuke(ctx context.Context, identifiers []string) error {
	if err := nfrp.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

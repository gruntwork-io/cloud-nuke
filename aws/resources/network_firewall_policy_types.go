package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type NetworkFirewallPolicyAPI interface {
	ListFirewallPolicies(ctx context.Context, params *networkfirewall.ListFirewallPoliciesInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallPoliciesOutput, error)
	DescribeFirewallPolicy(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallPolicyOutput, error)
	DeleteFirewallPolicy(ctx context.Context, params *networkfirewall.DeleteFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallPolicyOutput, error)
}

type NetworkFirewallPolicy struct {
	BaseAwsResource
	Client      NetworkFirewallPolicyAPI
	Region      string
	Identifiers []string
}

func (nfw *NetworkFirewallPolicy) Init(cfg aws.Config) {
	nfw.Client = networkfirewall.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (nfw *NetworkFirewallPolicy) ResourceName() string {
	return "network-firewall-policy"
}

func (nfw *NetworkFirewallPolicy) ResourceIdentifiers() []string {
	return nfw.Identifiers
}

func (nfw *NetworkFirewallPolicy) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (nfw *NetworkFirewallPolicy) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkFirewallPolicy
}

func (nfw *NetworkFirewallPolicy) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nfw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nfw.Identifiers = aws.ToStringSlice(identifiers)
	return nfw.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (nfw *NetworkFirewallPolicy) Nuke(identifiers []string) error {
	if err := nfw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

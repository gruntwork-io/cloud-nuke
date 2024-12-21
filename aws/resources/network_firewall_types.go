package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type NetworkFirewallAPI interface {
	ListFirewalls(ctx context.Context, params *networkfirewall.ListFirewallsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallsOutput, error)
	DescribeFirewall(ctx context.Context, params *networkfirewall.DescribeFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallOutput, error)
	DeleteFirewall(ctx context.Context, params *networkfirewall.DeleteFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallOutput, error)
}

type NetworkFirewall struct {
	BaseAwsResource
	Client      NetworkFirewallAPI
	Region      string
	Identifiers []string
}

func (nfw *NetworkFirewall) InitV2(cfg aws.Config) {
	nfw.Client = networkfirewall.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (nfw *NetworkFirewall) ResourceName() string {
	return "network-firewall"
}

func (nfw *NetworkFirewall) ResourceIdentifiers() []string {
	return nfw.Identifiers
}

func (nfw *NetworkFirewall) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (nfw *NetworkFirewall) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkFirewall
}

func (nfw *NetworkFirewall) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nfw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nfw.Identifiers = aws.ToStringSlice(identifiers)
	return nfw.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (nfw *NetworkFirewall) Nuke(identifiers []string) error {
	if err := nfw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type NetworkFirewallTLSConfigAPI interface {
	ListTLSInspectionConfigurations(ctx context.Context, params *networkfirewall.ListTLSInspectionConfigurationsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListTLSInspectionConfigurationsOutput, error)
	DescribeTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DescribeTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeTLSInspectionConfigurationOutput, error)
	DeleteTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DeleteTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteTLSInspectionConfigurationOutput, error)
}

type NetworkFirewallTLSConfig struct {
	BaseAwsResource
	Client      NetworkFirewallTLSConfigAPI
	Region      string
	Identifiers []string
}

func (nftc *NetworkFirewallTLSConfig) Init(cfg aws.Config) {
	nftc.Client = networkfirewall.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (nftc *NetworkFirewallTLSConfig) ResourceName() string {
	return "network-firewall-tls-config"
}

func (nftc *NetworkFirewallTLSConfig) ResourceIdentifiers() []string {
	return nftc.Identifiers
}

func (nftc *NetworkFirewallTLSConfig) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (nftc *NetworkFirewallTLSConfig) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkFirewallTLSConfig
}

func (nftc *NetworkFirewallTLSConfig) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nftc.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nftc.Identifiers = aws.ToStringSlice(identifiers)
	return nftc.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (nftc *NetworkFirewallTLSConfig) Nuke(identifiers []string) error {
	if err := nftc.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

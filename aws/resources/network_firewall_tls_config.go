package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// NetworkFirewallTLSConfigAPI defines the interface for Network Firewall TLS Config operations.
type NetworkFirewallTLSConfigAPI interface {
	ListTLSInspectionConfigurations(ctx context.Context, params *networkfirewall.ListTLSInspectionConfigurationsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListTLSInspectionConfigurationsOutput, error)
	DescribeTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DescribeTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeTLSInspectionConfigurationOutput, error)
	DeleteTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DeleteTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteTLSInspectionConfigurationOutput, error)
}

// NewNetworkFirewallTLSConfig creates a new Network Firewall TLS Config resource.
func NewNetworkFirewallTLSConfig() AwsResource {
	return NewAwsResource(&resource.Resource[NetworkFirewallTLSConfigAPI]{
		ResourceTypeName: "network-firewall-tls-config",
		BatchSize:        maxBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[NetworkFirewallTLSConfigAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = networkfirewall.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.NetworkFirewallTLSConfig
		},
		Lister: listNetworkFirewallTLSConfigs,
		Nuker:  resource.SimpleBatchDeleter(deleteNetworkFirewallTLSConfig),
	})
}

// listNetworkFirewallTLSConfigs retrieves all TLS inspection configurations that match the config filters.
func listNetworkFirewallTLSConfigs(ctx context.Context, client NetworkFirewallTLSConfigAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := networkfirewall.NewListTLSInspectionConfigurationsPaginator(client, &networkfirewall.ListTLSInspectionConfigurationsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, tlsConfig := range page.TLSInspectionConfigurations {
			output, err := client.DescribeTLSInspectionConfiguration(ctx, &networkfirewall.DescribeTLSInspectionConfigurationInput{
				TLSInspectionConfigurationArn: tlsConfig.Arn,
			})
			if err != nil {
				logging.Debugf("[Failed] Unable to describe TLS inspection configuration %s: %s", aws.ToString(tlsConfig.Name), err)
				return nil, errors.WithStackTrace(err)
			}

			if output.TLSInspectionConfigurationResponse == nil {
				logging.Debugf("[Skip] No TLS inspection configuration response for %s", aws.ToString(tlsConfig.Name))
				continue
			}

			tags := util.ConvertNetworkFirewallTagsToMap(output.TLSInspectionConfigurationResponse.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, tlsConfig.Arn, tags)
			if err != nil {
				logging.Debugf("[Failed] Unable to get first seen time for %s: %s", aws.ToString(tlsConfig.Name), err)
				return nil, errors.WithStackTrace(err)
			}

			// Use Name tag if available, otherwise use the configuration name
			identifierName := aws.ToString(output.TLSInspectionConfigurationResponse.TLSInspectionConfigurationName)
			if nameTag, ok := tags["Name"]; ok {
				identifierName = nameTag
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: &identifierName,
				Tags: tags,
				Time: firstSeenTime,
			}) {
				identifiers = append(identifiers, tlsConfig.Name)
			}
		}
	}

	return identifiers, nil
}

// deleteNetworkFirewallTLSConfig deletes a single TLS inspection configuration.
func deleteNetworkFirewallTLSConfig(ctx context.Context, client NetworkFirewallTLSConfigAPI, id *string) error {
	_, err := client.DeleteTLSInspectionConfiguration(ctx, &networkfirewall.DeleteTLSInspectionConfigurationInput{
		TLSInspectionConfigurationName: id,
	})
	return err
}

package resources

import (
	"context"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeNetworkFirewallTLSConfig(tlsconfig *networkfirewall.TLSInspectionConfigurationResponse, firstSeenTime *time.Time, configObj config.Config) bool {

	var identifierName string
	tags := util.ConvertNetworkFirewallTagsToMap(tlsconfig.Tags)

	identifierName = awsgo.StringValue(tlsconfig.TLSInspectionConfigurationName) // set the default
	if v, ok := tags["Name"]; ok {
		identifierName = v
	}

	return configObj.NetworkFirewallTLSConfig.ShouldInclude(config.ResourceValue{
		Name: &identifierName,
		Tags: tags,
		Time: firstSeenTime,
	})
}

func (nftc *NetworkFirewallTLSConfig) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var (
		identifiers   []*string
		firstSeenTime *time.Time
		err           error
	)

	meta, err := nftc.Client.ListTLSInspectionConfigurations(nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, tlsconfig := range meta.TLSInspectionConfigurations {
		output, err := nftc.Client.DescribeTLSInspectionConfiguration(&networkfirewall.DescribeTLSInspectionConfigurationInput{
			TLSInspectionConfigurationArn: tlsconfig.Arn,
		})
		if err != nil {
			logging.Errorf("[Failed] to describe the firewall TLS inspection configuation %s", awsgo.StringValue(tlsconfig.Name))
			return nil, errors.WithStackTrace(err)
		}

		if output.TLSInspectionConfigurationResponse == nil {
			logging.Errorf("[Failed] no firewall TLS inspection configuation information found for %s", awsgo.StringValue(tlsconfig.Name))
			continue
		}

		firstSeenTime, err = util.GetOrCreateFirstSeen(c, nftc.Client, tlsconfig.Arn, util.ConvertNetworkFirewallTagsToMap(output.TLSInspectionConfigurationResponse.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if shouldIncludeNetworkFirewallTLSConfig(output.TLSInspectionConfigurationResponse, firstSeenTime, configObj) {
			identifiers = append(identifiers, tlsconfig.Name)
		}
	}

	return identifiers, nil
}

func (nftc *NetworkFirewallTLSConfig) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Network Firewall TLS inspection configurations to nuke in region %s", nftc.Region)
		return nil
	}

	logging.Debugf("Deleting Network firewall TLS inspection configurations in region %s", nftc.Region)
	var deleted []*string

	for _, id := range identifiers {
		_, err := nftc.Client.DeleteTLSInspectionConfiguration(&networkfirewall.DeleteTLSInspectionConfigurationInput{
			TLSInspectionConfigurationName: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Network Firewall TLS inspection configurations",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deleted = append(deleted, id)
		}
	}

	logging.Debugf("[OK] %d Network Firewall TLS inspection configurations(s) deleted in %s", len(deleted), nftc.Region)

	return nil
}

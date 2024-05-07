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

func (nfw *NetworkFirewallTLSConfig) setFirstSeenTag(resource *networkfirewall.TLSInspectionConfigurationResponse, value time.Time) error {
	_, err := nfw.Client.TagResource(&networkfirewall.TagResourceInput{
		ResourceArn: resource.TLSInspectionConfigurationArn,
		Tags: []*networkfirewall.Tag{
			{
				Key:   awsgo.String(util.FirstSeenTagKey),
				Value: awsgo.String(util.FormatTimestamp(value)),
			},
		},
	})
	return errors.WithStackTrace(err)
}

func (nfw *NetworkFirewallTLSConfig) getFirstSeenTag(resource *networkfirewall.TLSInspectionConfigurationResponse) (*time.Time, error) {
	for _, tag := range resource.Tags {
		if util.IsFirstSeenTag(tag.Key) {
			firstSeenTime, err := util.ParseTimestamp(tag.Value)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return nil, nil
}

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

func (nftc *NetworkFirewallTLSConfig) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
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

		// check first seen tag
		firstSeenTime, err := nftc.getFirstSeenTag(output.TLSInspectionConfigurationResponse)
		if err != nil {
			logging.Errorf(
				"Unable to retrieve tags for TLS inspection configurations: %s, with error: %s", awsgo.StringValue(tlsconfig.Name), err)
			continue
		}

		// if the first seen tag is not there, then create one
		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := nftc.setFirstSeenTag(output.TLSInspectionConfigurationResponse, time.Now().UTC()); err != nil {
				logging.Errorf(
					"Unable to apply first seen tag TLS inspection configurations: %s, with error: %s", awsgo.StringValue(tlsconfig.Name), err)
				continue
			}
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

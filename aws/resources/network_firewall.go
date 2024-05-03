package resources

import (
	"context"
	"slices"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeNetworkFirewall(firewall *networkfirewall.Firewall, firstSeenTime *time.Time, configObj config.Config) bool {
	var identifierName string
	tags := util.ConvertNetworkFirewallTagsToMap(firewall.Tags)

	identifierName = awsgo.StringValue(firewall.FirewallName) // set the default
	if v, ok := tags["Name"]; ok {
		identifierName = v
	}

	return configObj.NetworkFirewall.ShouldInclude(config.ResourceValue{
		Name: &identifierName,
		Tags: tags,
		Time: firstSeenTime,
	})
}

func (nfw *NetworkFirewall) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
	var firstSeenTime *time.Time
	var err error

	metaOutput, err := nfw.Client.ListFirewalls(nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var deleteprotected []string
	// describe the firewalls to get more info
	for _, firewall := range metaOutput.Firewalls {
		output, err := nfw.Client.DescribeFirewall(&networkfirewall.DescribeFirewallInput{
			FirewallArn: firewall.FirewallArn,
		})
		if err != nil {
			logging.Errorf("[Failed] to describe the firewall %s", awsgo.StringValue(firewall.FirewallArn))
			return nil, errors.WithStackTrace(err)
		}

		if output.Firewall == nil {
			logging.Errorf("[Failed] no firewall information found for %s", awsgo.StringValue(firewall.FirewallArn))
			continue
		}

		firstSeenTime, err = util.GetOrCreateFirstSeen(c, nfw.Client, firewall.FirewallArn, util.ConvertNetworkFirewallTagsToMap(output.Firewall.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		// check the resource is delete protected
		if awsgo.BoolValue(output.Firewall.DeleteProtection) {
			deleteprotected = append(deleteprotected, awsgo.StringValue(firewall.FirewallName))
		}

		if shouldIncludeNetworkFirewall(output.Firewall, firstSeenTime, configObj) {
			identifiers = append(identifiers, firewall.FirewallName)
		}
	}

	nfw.VerifyNukablePermissions(identifiers, func(id *string) error {
		// check the resource is enabled delete protection
		if slices.Contains(deleteprotected, awsgo.StringValue(id)) {
			return util.ErrDeleteProtectionEnabled
		}
		return nil
	})

	return identifiers, nil
}

func (nfw *NetworkFirewall) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Network Firewalls to nuke in region %s", nfw.Region)
		return nil
	}

	logging.Debugf("Deleting Network firewalls in region %s", nfw.Region)
	var deleted []*string

	for _, id := range identifiers {
		if nukable, err := nfw.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		_, err := nfw.Client.DeleteFirewall(&networkfirewall.DeleteFirewallInput{
			FirewallName: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Network Firewall",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deleted = append(deleted, id)
		}
	}

	logging.Debugf("[OK] %d Network Firewall(s) deleted in %s", len(deleted), nfw.Region)

	return nil
}

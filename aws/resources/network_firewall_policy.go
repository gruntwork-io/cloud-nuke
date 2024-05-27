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

func shouldIncludeNetworkFirewallPolicy(firewall *networkfirewall.FirewallPolicyResponse, firstSeenTime *time.Time, configObj config.Config) bool {
	// if the firewall policy has any attachments, then we can't remove that policy
	if awsgo.Int64Value(firewall.NumberOfAssociations) > 0 {
		logging.Debugf("[Skipping] the policy %s is still in use", awsgo.StringValue(firewall.FirewallPolicyName))
		return false
	}

	var identifierName string
	tags := util.ConvertNetworkFirewallTagsToMap(firewall.Tags)

	if v, ok := tags["Name"]; ok {
		identifierName = v
	}
	return configObj.NetworkFirewallPolicy.ShouldInclude(config.ResourceValue{
		Name: &identifierName,
		Tags: tags,
		Time: firstSeenTime,
	})
}

func (nfw *NetworkFirewallPolicy) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var (
		identifiers   []*string
		firstSeenTime *time.Time
		err           error
	)

	metaOutput, err := nfw.Client.ListFirewallPoliciesWithContext(nfw.Context, nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, policy := range metaOutput.FirewallPolicies {

		output, err := nfw.Client.DescribeFirewallPolicyWithContext(nfw.Context, &networkfirewall.DescribeFirewallPolicyInput{
			FirewallPolicyArn: policy.Arn,
		})
		if err != nil {
			logging.Errorf("[Failed] to describe the firewall policy %s", awsgo.StringValue(policy.Name))
			return nil, errors.WithStackTrace(err)
		}

		if output.FirewallPolicyResponse == nil {
			logging.Errorf("[Failed] no firewall policy information found for %s", awsgo.StringValue(policy.Name))
			continue
		}

		firstSeenTime, err = util.GetOrCreateFirstSeen(c, nfw.Client, policy.Arn, util.ConvertNetworkFirewallTagsToMap(output.FirewallPolicyResponse.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if shouldIncludeNetworkFirewallPolicy(output.FirewallPolicyResponse, firstSeenTime, configObj) {
			identifiers = append(identifiers, policy.Name)
		}
	}

	return identifiers, nil
}

func (nfw *NetworkFirewallPolicy) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Network Firewall policy to nuke in region %s", nfw.Region)
		return nil
	}

	logging.Debugf("Deleting Network firewall policy in region %s", nfw.Region)
	var deleted []*string

	for _, id := range identifiers {
		_, err := nfw.Client.DeleteFirewallPolicyWithContext(nfw.Context, &networkfirewall.DeleteFirewallPolicyInput{
			FirewallPolicyName: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Network Firewall policy",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deleted = append(deleted, id)
		}
	}

	logging.Debugf("[OK] %d Network Policy(s) deleted in %s", len(deleted), nfw.Region)

	return nil
}

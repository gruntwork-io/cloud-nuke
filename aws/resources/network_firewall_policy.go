package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeNetworkFirewallPolicy(firewall *types.FirewallPolicyResponse, firstSeenTime *time.Time, configObj config.Config) bool {
	// if the firewall policy has any attachments, then we can't remove that policy
	if aws.ToInt32(firewall.NumberOfAssociations) > 0 {
		logging.Debugf("[Skipping] the policy %s is still in use", aws.ToString(firewall.FirewallPolicyName))
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

	metaOutput, err := nfw.Client.ListFirewallPolicies(nfw.Context, nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, policy := range metaOutput.FirewallPolicies {

		output, err := nfw.Client.DescribeFirewallPolicy(nfw.Context, &networkfirewall.DescribeFirewallPolicyInput{
			FirewallPolicyArn: policy.Arn,
		})
		if err != nil {
			logging.Errorf("[Failed] to describe the firewall policy %s", aws.ToString(policy.Name))
			return nil, errors.WithStackTrace(err)
		}

		if output.FirewallPolicyResponse == nil {
			logging.Errorf("[Failed] no firewall policy information found for %s", aws.ToString(policy.Name))
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
		_, err := nfw.Client.DeleteFirewallPolicy(nfw.Context, &networkfirewall.DeleteFirewallPolicyInput{
			FirewallPolicyName: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
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

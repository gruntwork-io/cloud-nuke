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

func (nfw *NetworkFirewallPolicy) setFirstSeenTag(resource *networkfirewall.FirewallPolicyResponse, value time.Time) error {
	_, err := nfw.Client.TagResource(&networkfirewall.TagResourceInput{
		ResourceArn: resource.FirewallPolicyArn,
		Tags: []*networkfirewall.Tag{
			{
				Key:   awsgo.String(util.FirstSeenTagKey),
				Value: awsgo.String(util.FormatTimestamp(value)),
			},
		},
	})
	return errors.WithStackTrace(err)
}

func (nfw *NetworkFirewallPolicy) getFirstSeenTag(resource *networkfirewall.FirewallPolicyResponse) (*time.Time, error) {
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

func (nfw *NetworkFirewallPolicy) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string

	metaOutput, err := nfw.Client.ListFirewallPolicies(nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, policy := range metaOutput.FirewallPolicies {

		output, err := nfw.Client.DescribeFirewallPolicy(&networkfirewall.DescribeFirewallPolicyInput{
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

		// check first seen tag
		firstSeenTime, err := nfw.getFirstSeenTag(output.FirewallPolicyResponse)
		if err != nil {
			logging.Errorf(
				"Unable to retrieve tags for Rule group: %s, with error: %s", awsgo.StringValue(policy.Name), err)
			continue
		}

		// if the first seen tag is not there, then create one
		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := nfw.setFirstSeenTag(output.FirewallPolicyResponse, time.Now().UTC()); err != nil {
				logging.Errorf(
					"Unable to apply first seen tag Rule group: %s, with error: %s", awsgo.StringValue(policy.Name), err)
				continue
			}
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
		_, err := nfw.Client.DeleteFirewallPolicy(&networkfirewall.DeleteFirewallPolicyInput{
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

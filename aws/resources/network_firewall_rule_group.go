package resources

import (
	"context"
	"fmt"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (nfw *NetworkFirewallRuleGroup) setFirstSeenTag(resource *networkfirewall.RuleGroupResponse, value time.Time) error {
	_, err := nfw.Client.TagResource(&networkfirewall.TagResourceInput{
		ResourceArn: resource.RuleGroupArn,
		Tags: []*networkfirewall.Tag{
			{
				Key:   awsgo.String(util.FirstSeenTagKey),
				Value: awsgo.String(util.FormatTimestamp(value)),
			},
		},
	})
	return errors.WithStackTrace(err)
}

func (nfw *NetworkFirewallRuleGroup) getFirstSeenTag(resource *networkfirewall.RuleGroupResponse) (*time.Time, error) {
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

func shouldIncludeNetworkFirewallRuleGroup(group *networkfirewall.RuleGroupResponse, firstSeenTime *time.Time, configObj config.Config) bool {
	// if the firewall policy has any attachments, then we can't remove that policy
	if awsgo.Int64Value(group.NumberOfAssociations) > 0 {
		logging.Debugf("[Skipping] the rule group %s is still in use", awsgo.StringValue(group.RuleGroupName))
		return false
	}

	var identifierName string
	tags := util.ConvertNetworkFirewallTagsToMap(group.Tags)

	identifierName = awsgo.StringValue(group.RuleGroupName) // set the default
	if v, ok := tags["Name"]; ok {
		identifierName = v
	}

	return configObj.NetworkFirewallRuleGroup.ShouldInclude(config.ResourceValue{
		Name: &identifierName,
		Tags: tags,
		Time: firstSeenTime,
	})
}

func (nfw *NetworkFirewallRuleGroup) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string

	meta, err := nfw.Client.ListRuleGroups(nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, group := range meta.RuleGroups {
		output, err := nfw.Client.DescribeRuleGroup(&networkfirewall.DescribeRuleGroupInput{
			RuleGroupArn: group.Arn,
		})
		if err != nil {
			logging.Errorf("[Failed] to describe the firewall rule group %s", awsgo.StringValue(group.Name))
			return nil, errors.WithStackTrace(err)
		}

		if output.RuleGroupResponse == nil {
			logging.Errorf("[Failed] no firewall rule group information found for %s", awsgo.StringValue(group.Name))
			continue
		}

		// check first seen tag
		firstSeenTime, err := nfw.getFirstSeenTag(output.RuleGroupResponse)
		if err != nil {
			logging.Errorf(
				"Unable to retrieve tags for Rule group: %s, with error: %s", awsgo.StringValue(group.Name), err)
			continue
		}

		// if the first seen tag is not there, then create one
		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := nfw.setFirstSeenTag(output.RuleGroupResponse, time.Now().UTC()); err != nil {
				logging.Errorf(
					"Unable to apply first seen tag Rule group: %s, with error: %s", awsgo.StringValue(group.Name), err)
				continue
			}
		}

		if shouldIncludeNetworkFirewallRuleGroup(output.RuleGroupResponse, firstSeenTime, configObj) {
			identifiers = append(identifiers, group.Name)

			raw := awsgo.StringValue(group.Name)
			nfw.RuleGroups[raw] = RuleGroup{
				Name: output.RuleGroupResponse.RuleGroupName,
				Type: output.RuleGroupResponse.Type,
			}
		}
	}

	return identifiers, nil
}

func (nfw *NetworkFirewallRuleGroup) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Network Firewall rule group to nuke in region %s", nfw.Region)
		return nil
	}

	logging.Debugf("Deleting Network firewall rule group in region %s", nfw.Region)
	var deleted []*string

	for _, id := range identifiers {
		// check and get the type for this identifier
		group, ok := nfw.RuleGroups[awsgo.StringValue(id)]
		if !ok {
			logging.Errorf("couldn't find the rule group type for %s", awsgo.StringValue(id))
			return fmt.Errorf("couldn't find the rule group type for %s", awsgo.StringValue(id))
		}

		// delete the rule group
		_, err := nfw.Client.DeleteRuleGroup(&networkfirewall.DeleteRuleGroupInput{
			RuleGroupName: id,
			Type:          group.Type,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Network Firewall Rule group",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deleted = append(deleted, id)
		}
	}

	logging.Debugf("[OK] %d Network Firewall Rule group(s) deleted in %s", len(deleted), nfw.Region)

	return nil
}

package resources

import (
	"context"
	"fmt"
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

func shouldIncludeNetworkFirewallRuleGroup(group *types.RuleGroupResponse, firstSeenTime *time.Time, configObj config.Config) bool {
	// if the firewall policy has any attachments, then we can't remove that policy
	if aws.ToInt32(group.NumberOfAssociations) > 0 {
		logging.Debugf("[Skipping] the rule group %s is still in use", aws.ToString(group.RuleGroupName))
		return false
	}

	var identifierName string
	tags := util.ConvertNetworkFirewallTagsToMap(group.Tags)

	identifierName = aws.ToString(group.RuleGroupName) // set the default
	if v, ok := tags["Name"]; ok {
		identifierName = v
	}

	return configObj.NetworkFirewallRuleGroup.ShouldInclude(config.ResourceValue{
		Name: &identifierName,
		Tags: tags,
		Time: firstSeenTime,
	})
}

func (nfrg *NetworkFirewallRuleGroup) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var (
		identifiers   []*string
		firstSeenTime *time.Time
		err           error
	)

	meta, err := nfrg.Client.ListRuleGroups(nfrg.Context, nil)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, group := range meta.RuleGroups {
		output, err := nfrg.Client.DescribeRuleGroup(nfrg.Context, &networkfirewall.DescribeRuleGroupInput{
			RuleGroupArn: group.Arn,
		})
		if err != nil {
			logging.Errorf("[Failed] to describe the firewall rule group %s", aws.ToString(group.Name))
			return nil, errors.WithStackTrace(err)
		}

		if output.RuleGroupResponse == nil {
			logging.Errorf("[Failed] no firewall rule group information found for %s", aws.ToString(group.Name))
			continue
		}

		firstSeenTime, err = util.GetOrCreateFirstSeen(c, nfrg.Client, group.Arn, util.ConvertNetworkFirewallTagsToMap(output.RuleGroupResponse.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if shouldIncludeNetworkFirewallRuleGroup(output.RuleGroupResponse, firstSeenTime, configObj) {
			identifiers = append(identifiers, group.Name)

			raw := aws.ToString(group.Name)
			nfrg.RuleGroups[raw] = RuleGroup{
				Name: output.RuleGroupResponse.RuleGroupName,
				Type: aws.String(string(output.RuleGroupResponse.Type)),
			}
		}
	}

	return identifiers, nil
}

func (nfrg *NetworkFirewallRuleGroup) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Network Firewall rule group to nuke in region %s", nfrg.Region)
		return nil
	}

	logging.Debugf("Deleting Network firewall rule group in region %s", nfrg.Region)
	var deleted []*string

	for _, id := range identifiers {
		// check and get the type for this identifier
		group, ok := nfrg.RuleGroups[aws.ToString(id)]
		if !ok {
			logging.Errorf("couldn't find the rule group type for %s", aws.ToString(id))
			return fmt.Errorf("couldn't find the rule group type for %s", aws.ToString(id))
		}

		// delete the rule group
		_, err := nfrg.Client.DeleteRuleGroup(nfrg.Context, &networkfirewall.DeleteRuleGroupInput{
			RuleGroupName: id,
			Type:          types.RuleGroupType(aws.ToString(group.Type)),
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
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

	logging.Debugf("[OK] %d Network Firewall Rule group(s) deleted in %s", len(deleted), nfrg.Region)

	return nil
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// The Network Firewall service supports only one type of resource-based policy called a resource policy, which is attached to a shared firewall policy or rule group
// References :
// - https://docs.aws.amazon.com/network-firewall/latest/developerguide/security_iam_resource-based-policy-examples.html
// - https://docs.aws.amazon.com/network-firewall/latest/developerguide/sharing.html
// - https://docs.aws.amazon.com/ram/latest/userguide/what-is.html
func (nfrp *NetworkFirewallResourcePolicy) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string

	var resourceArns []*string
	// list the firewall policies and rule group

	{
		policyMeta, err := nfrp.Client.ListFirewallPolicies(nfrp.Context, nil)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, policy := range policyMeta.FirewallPolicies {
			resourceArns = append(resourceArns, policy.Arn)
		}
		groupMeta, err := nfrp.Client.ListRuleGroups(nfrp.Context, nil)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, group := range groupMeta.RuleGroups {
			resourceArns = append(resourceArns, group.Arn)
		}
	}

	// get the resource policies attached on these arns
	for _, arn := range resourceArns {
		output, err := nfrp.Client.DescribeResourcePolicy(nfrp.Context, &networkfirewall.DescribeResourcePolicyInput{
			ResourceArn: arn,
		})
		if err != nil && util.TransformAWSError(err) != util.ErrResourceNotFoundException {
			return nil, errors.WithStackTrace(err)
		}

		// if the policy exists for a resource
		if output != nil && output.Policy != nil {
			if configObj.NetworkFirewallResourcePolicy.ShouldInclude(config.ResourceValue{
				Name: arn,
			}) {
				identifiers = append(identifiers, arn)
			}
		}
	}
	return identifiers, nil
}

func (nfrp *NetworkFirewallResourcePolicy) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Network Firewall resource policy to nuke in region %s", nfrp.Region)
		return nil
	}

	logging.Debugf("Deleting Network firewall resource policy in region %s", nfrp.Region)
	var deleted []*string

	for _, id := range identifiers {
		_, err := nfrp.Client.DeleteResourcePolicy(nfrp.Context, &networkfirewall.DeleteResourcePolicyInput{
			ResourceArn: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: "Network Firewall Resource policy",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deleted = append(deleted, id)
		}
	}

	logging.Debugf("[OK] %d Network Resource Policy(s) deleted in %s", len(deleted), nfrp.Region)

	return nil
}

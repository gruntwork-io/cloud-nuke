package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/service/networkfirewall/networkfirewalliface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type RuleGroup struct {
	Name, Type *string
}

type NetworkFirewallRuleGroup struct {
	BaseAwsResource
	Client      networkfirewalliface.NetworkFirewallAPI
	Region      string
	Identifiers []string
	// Note: It is mandatory to pass the rule type while nuking it.
	// This map can be used to store information about a rule group name as key and value.
	// When invoking the nuke method, information about a rule can be easily retrieved without making another API request.
	RuleGroups map[string]RuleGroup
}

func (nfrg *NetworkFirewallRuleGroup) Init(session *session.Session) {
	nfrg.Client = networkfirewall.New(session)
	nfrg.RuleGroups = make(map[string]RuleGroup, 0)
}

// ResourceName - the simple name of the aws resource
func (nfrg *NetworkFirewallRuleGroup) ResourceName() string {
	return "network-firewall-rule-group"
}

func (nfrg *NetworkFirewallRuleGroup) ResourceIdentifiers() []string {
	return nfrg.Identifiers
}

func (nfrg *NetworkFirewallRuleGroup) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (nfrg *NetworkFirewallRuleGroup) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkFirewallRuleGroup
}

func (nfrg *NetworkFirewallRuleGroup) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nfrg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nfrg.Identifiers = awsgo.StringValueSlice(identifiers)
	return nfrg.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (nfrg *NetworkFirewallRuleGroup) Nuke(identifiers []string) error {
	if err := nfrg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

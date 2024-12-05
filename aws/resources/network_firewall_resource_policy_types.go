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

type NetworkFirewallResourcePolicy struct {
	BaseAwsResource
	Client      networkfirewalliface.NetworkFirewallAPI
	Region      string
	Identifiers []string
}

func (nfrp *NetworkFirewallResourcePolicy) Init(session *session.Session) {
	nfrp.Client = networkfirewall.New(session)
}

// ResourceName - the simple name of the aws resource
func (nfrp *NetworkFirewallResourcePolicy) ResourceName() string {
	return "network-firewall-resource-policy"
}

func (nfrp *NetworkFirewallResourcePolicy) ResourceIdentifiers() []string {
	return nfrp.Identifiers
}

func (nfrp *NetworkFirewallResourcePolicy) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (nfrp *NetworkFirewallResourcePolicy) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkFirewallResourcePolicy
}

func (nfrp *NetworkFirewallResourcePolicy) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nfrp.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nfrp.Identifiers = awsgo.StringValueSlice(identifiers)
	return nfrp.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (nfrp *NetworkFirewallResourcePolicy) Nuke(identifiers []string) error {
	if err := nfrp.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

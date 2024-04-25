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

type NetworkFirewallTLSConfig struct {
	BaseAwsResource
	Client      networkfirewalliface.NetworkFirewallAPI
	Region      string
	Identifiers []string
}

func (nftc *NetworkFirewallTLSConfig) Init(session *session.Session) {
	nftc.Client = networkfirewall.New(session)
}

// ResourceName - the simple name of the aws resource
func (nftc *NetworkFirewallTLSConfig) ResourceName() string {
	return "network-firewall-tls-config"
}

func (nftc *NetworkFirewallTLSConfig) ResourceIdentifiers() []string {
	return nftc.Identifiers
}

func (nftc *NetworkFirewallTLSConfig) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (nftc *NetworkFirewallTLSConfig) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nftc.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nftc.Identifiers = awsgo.StringValueSlice(identifiers)
	return nftc.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (nftc *NetworkFirewallTLSConfig) Nuke(identifiers []string) error {
	if err := nftc.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

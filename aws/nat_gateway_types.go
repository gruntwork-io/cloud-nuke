package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// NatGateways - represents all AWS secrets manager secrets that should be deleted.
type NatGateways struct {
	Client        ec2iface.EC2API
	Region        string
	NatGatewayIDs []string
}

// ResourceName - the simple name of the aws resource
func (ngw NatGateways) ResourceName() string {
	return "nat-gateway"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (ngw NatGateways) ResourceIdentifiers() []string {
	return ngw.NatGatewayIDs
}

func (secret NatGateways) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

// Nuke - nuke 'em all!!!
func (ngw NatGateways) Nuke(session *session.Session, identifiers []string) error {
	if err := ngw.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

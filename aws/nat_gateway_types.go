package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// NATGateways - represents all NAT gateways
type NATGateways struct {
	NatGatewayIds []string
}

// ResourceName - the simple name of the aws resource
func (natGateways NATGateways) ResourceName() string {
	return "natgateway"
}

// ResourceIdentifiers - The instance ids of the eip addresses
func (natGateways NATGateways) ResourceIdentifiers() []string {
	return natGateways.NatGatewayIds
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (natGateways NATGateways) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (natGateways NATGateways) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllNatGateways(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type NatGatewayDeleteError struct{}

func (e NatGatewayDeleteError) Error() string {
	return "NAT Gateway was not deleted"
}

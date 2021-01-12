package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// TransitGateways - represents all transit gateways
type TransitGateways struct {
	Ids []string
}

// ResourceName - the simple name of the aws resource
func (tgw TransitGateways) ResourceName() string {
	return "transit-gateway"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw TransitGateways) MaxBatchSize() int {
	return 200
}

// ResourceIdentifiers - The Ids of the transit gateways
func (tgw TransitGateways) ResourceIdentifiers() []string {
	return tgw.Ids
}

// Nuke - nuke 'em all!!!
func (tgw TransitGateways) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllTransitGatewayInstances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// TransitGatewaysRouteTables - represents all transit gateways route tables
type TransitGatewaysRouteTables struct {
	Ids []string
}

// ResourceName - the simple name of the aws resource
func (tgw TransitGatewaysRouteTables) ResourceName() string {
	return "transit-gateway-route-table"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw TransitGatewaysRouteTables) MaxBatchSize() int {
	return 200
}

// ResourceIdentifiers - The arns of the transit gateways route tables
func (tgw TransitGatewaysRouteTables) ResourceIdentifiers() []string {
	return tgw.Ids
}

// Nuke - nuke 'em all!!!
func (tgw TransitGatewaysRouteTables) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllTransitGatewayRouteTables(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

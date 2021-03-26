package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// TransitGatewaysVpcAttachment - represents all transit gateways vpc attachments
type TransitGatewaysVpcAttachment struct {
	Ids []string
}

// ResourceName - the simple name of the aws resource
func (tgw TransitGatewaysVpcAttachment) ResourceName() string {
	return "transit-gateway-attachment"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw TransitGatewaysVpcAttachment) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the transit gateways
func (tgw TransitGatewaysVpcAttachment) ResourceIdentifiers() []string {
	return tgw.Ids
}

// Nuke - nuke 'em all!!!
func (tgw TransitGatewaysVpcAttachment) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllTransitGatewayVpcAttachments(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

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
	return maxBatchSize
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
	return maxBatchSize
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

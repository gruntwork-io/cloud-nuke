package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// TransitGatewayVpcAttachment - represents all transit gateways vpc attachments
type TransitGatewayVpcAttachment struct {
	Client ec2iface.EC2API
	Region string
	Ids    []string
}

// ResourceName - the simple name of the aws resource
func (tgw TransitGatewayVpcAttachment) ResourceName() string {
	return "transit-gateway-attachment"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw TransitGatewayVpcAttachment) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the transit gateways
func (tgw TransitGatewayVpcAttachment) ResourceIdentifiers() []string {
	return tgw.Ids
}

// Nuke - nuke 'em all!!!
func (tgw TransitGatewayVpcAttachment) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllTransitGatewayVpcAttachments(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// TransitGatewayRouteTable - represents all transit gateways route tables
type TransitGatewayRouteTable struct {
	Client ec2iface.EC2API
	Region string
	Ids    []string
}

// ResourceName - the simple name of the aws resource
func (tgw TransitGatewayRouteTable) ResourceName() string {
	return "transit-gateway-route-table"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw TransitGatewayRouteTable) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The arns of the transit gateways route tables
func (tgw TransitGatewayRouteTable) ResourceIdentifiers() []string {
	return tgw.Ids
}

// Nuke - nuke 'em all!!!
func (tgw TransitGatewayRouteTable) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllTransitGatewayRouteTables(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// TransitGateway - represents all transit gateways
type TransitGateway struct {
	Client ec2iface.EC2API
	Region string
	Ids    []string
}

// ResourceName - the simple name of the aws resource
func (tgw TransitGateway) ResourceName() string {
	return "transit-gateway"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw TransitGateway) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the transit gateways
func (tgw TransitGateway) ResourceIdentifiers() []string {
	return tgw.Ids
}

// Nuke - nuke 'em all!!!
func (tgw TransitGateway) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllTransitGatewayInstances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

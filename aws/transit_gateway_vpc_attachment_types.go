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
	return 200
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

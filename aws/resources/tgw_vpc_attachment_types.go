package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type TGWVpcAPI interface {
	DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error)
	DescribeTransitGatewayVpcAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayVpcAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error)
}

// TransitGatewaysVpcAttachment represents all transit gateway VPC attachments.
type TransitGatewaysVpcAttachment struct {
	BaseAwsResource
	Client TGWVpcAPI
	Region string
	Ids    []string
}

func (tgw *TransitGatewaysVpcAttachment) InitV2(cfg aws.Config) {
	tgw.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (tgw *TransitGatewaysVpcAttachment) ResourceName() string {
	return "transit-gateway-attachment"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw *TransitGatewaysVpcAttachment) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the transit gateways
func (tgw *TransitGatewaysVpcAttachment) ResourceIdentifiers() []string {
	return tgw.Ids
}

func (tgw *TransitGatewaysVpcAttachment) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := tgw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	tgw.Ids = aws.ToStringSlice(identifiers)
	return tgw.Ids, nil
}

// Nuke - nuke 'em all!!!
func (tgw *TransitGatewaysVpcAttachment) Nuke(identifiers []string) error {
	if err := tgw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

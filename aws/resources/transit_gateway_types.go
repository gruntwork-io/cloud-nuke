package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type TransitGatewayAPI interface {
	DescribeTransitGateways(ctx context.Context, params *ec2.DescribeTransitGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewaysOutput, error)
	DeleteTransitGateway(ctx context.Context, params *ec2.DeleteTransitGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayOutput, error)
	DescribeTransitGatewayAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayAttachmentsOutput, error)
	DeleteTransitGatewayPeeringAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayPeeringAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error)
	DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error)
	DeleteTransitGatewayConnect(ctx context.Context, params *ec2.DeleteTransitGatewayConnectInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayConnectOutput, error)
}

// TransitGateways - represents all transit gateways
type TransitGateways struct {
	BaseAwsResource
	Client TransitGatewayAPI
	Region string
	Ids    []string
}

func (tgw *TransitGateways) Init(cfg aws.Config) {
	tgw.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (tgw *TransitGateways) ResourceName() string {
	return "transit-gateway"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw *TransitGateways) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the transit gateways
func (tgw *TransitGateways) ResourceIdentifiers() []string {
	return tgw.Ids
}

func (tgw *TransitGateways) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := tgw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	tgw.Ids = aws.ToStringSlice(identifiers)
	return tgw.Ids, nil
}

// Nuke - nuke 'em all!!!
func (tgw *TransitGateways) Nuke(identifiers []string) error {
	if err := tgw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

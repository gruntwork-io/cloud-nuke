package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type TransitGatewayPeeringAttachmentAPI interface {
	DescribeTransitGatewayPeeringAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayPeeringAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayPeeringAttachmentsOutput, error)
	DeleteTransitGatewayPeeringAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayPeeringAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error)
}

// TransitGatewayPeeringAttachment - represents all transit gateways peering attachment
type TransitGatewayPeeringAttachment struct {
	BaseAwsResource
	Client TransitGatewayPeeringAttachmentAPI
	Region string
	Ids    []string
}

func (tgpa *TransitGatewayPeeringAttachment) InitV2(cfg aws.Config) {
	tgpa.Client = ec2.NewFromConfig(cfg)
}

func (tgpa *TransitGatewayPeeringAttachment) IsUsingV2() bool { return true }

func (tgpa *TransitGatewayPeeringAttachment) ResourceName() string {
	return "transit-gateway-peering-attachment"
}

func (tgpa *TransitGatewayPeeringAttachment) MaxBatchSize() int {
	return maxBatchSize
}

func (tgpa *TransitGatewayPeeringAttachment) ResourceIdentifiers() []string {
	return tgpa.Ids
}

func (tgpa *TransitGatewayPeeringAttachment) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.TransitGatewayPeeringAttachment
}

func (tgpa *TransitGatewayPeeringAttachment) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := tgpa.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	tgpa.Ids = aws.ToStringSlice(identifiers)
	return tgpa.Ids, nil
}

func (tgpa *TransitGatewayPeeringAttachment) Nuke(identifiers []string) error {
	if err := tgpa.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

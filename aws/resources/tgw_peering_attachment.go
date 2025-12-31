package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// TransitGatewayPeeringAttachmentAPI defines the interface for Transit Gateway Peering Attachment operations.
type TransitGatewayPeeringAttachmentAPI interface {
	DescribeTransitGatewayPeeringAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayPeeringAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayPeeringAttachmentsOutput, error)
	DeleteTransitGatewayPeeringAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayPeeringAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error)
}

// NewTransitGatewayPeeringAttachment creates a new TransitGatewayPeeringAttachment resource using the generic resource pattern.
func NewTransitGatewayPeeringAttachment() AwsResource {
	return NewAwsResource(&resource.Resource[TransitGatewayPeeringAttachmentAPI]{
		ResourceTypeName: "transit-gateway-peering-attachment",
		BatchSize:        maxBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[TransitGatewayPeeringAttachmentAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.TransitGatewayPeeringAttachment
		},
		Lister: listTransitGatewayPeeringAttachments,
		Nuker:  resource.SimpleBatchDeleter(deleteTransitGatewayPeeringAttachment),
	})
}

// listTransitGatewayPeeringAttachments retrieves all Transit Gateway Peering Attachments that match the config filters.
func listTransitGatewayPeeringAttachments(ctx context.Context, client TransitGatewayPeeringAttachmentAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string

	paginator := ec2.NewDescribeTransitGatewayPeeringAttachmentsPaginator(client, &ec2.DescribeTransitGatewayPeeringAttachmentsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, attachment := range page.TransitGatewayPeeringAttachments {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: attachment.CreationTime,
			}) {
				ids = append(ids, attachment.TransitGatewayAttachmentId)
			}
		}
	}

	return ids, nil
}

// deleteTransitGatewayPeeringAttachment deletes a single Transit Gateway Peering Attachment.
func deleteTransitGatewayPeeringAttachment(ctx context.Context, client TransitGatewayPeeringAttachmentAPI, id *string) error {
	_, err := client.DeleteTransitGatewayPeeringAttachment(ctx, &ec2.DeleteTransitGatewayPeeringAttachmentInput{
		TransitGatewayAttachmentId: id,
	})
	return err
}

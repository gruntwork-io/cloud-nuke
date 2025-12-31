package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	goerror "github.com/go-errors/errors"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// TransitGatewaysVpcAttachmentAPI defines the interface for TransitGateway VPC Attachment operations.
type TransitGatewaysVpcAttachmentAPI interface {
	DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error)
	DescribeTransitGatewayVpcAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayVpcAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error)
}

// NewTransitGatewaysVpcAttachment creates a new TransitGatewaysVpcAttachment resource using the generic resource pattern.
func NewTransitGatewaysVpcAttachment() AwsResource {
	return NewAwsResource(&resource.Resource[TransitGatewaysVpcAttachmentAPI]{
		ResourceTypeName: "transit-gateway-attachment",
		BatchSize:        maxBatchSize,
		InitClient: func(r *resource.Resource[TransitGatewaysVpcAttachmentAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EC2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ec2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.TransitGatewaysVpcAttachment
		},
		Lister: listTransitGatewaysVpcAttachments,
		Nuker:  deleteTransitGatewaysVpcAttachments,
	})
}

// listTransitGatewaysVpcAttachments retrieves all Transit Gateway VPC Attachments that match the config filters.
func listTransitGatewaysVpcAttachments(ctx context.Context, client TransitGatewaysVpcAttachmentAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string
	params := &ec2.DescribeTransitGatewayVpcAttachmentsInput{}

	hasMorePages := true
	for hasMorePages {
		result, err := client.DescribeTransitGatewayVpcAttachments(ctx, params)
		if err != nil {
			logging.Debugf("[Transit Gateway] Failed to list transit gateway VPC attachments: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, tgwVpcAttachment := range result.TransitGatewayVpcAttachments {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: tgwVpcAttachment.CreationTime,
			}) && tgwVpcAttachment.State != "deleted" && tgwVpcAttachment.State != "deleting" {
				identifiers = append(identifiers, tgwVpcAttachment.TransitGatewayAttachmentId)
			}
		}

		params.NextToken = result.NextToken
		hasMorePages = params.NextToken != nil
	}

	return identifiers, nil
}

// deleteTransitGatewaysVpcAttachments deletes all Transit Gateway VPC Attachments.
func deleteTransitGatewaysVpcAttachments(ctx context.Context, client TransitGatewaysVpcAttachmentAPI, scope resource.Scope, resourceType string, ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Transit Gateway Vpc Attachments to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting all Transit Gateway Vpc Attachments in region %s", scope.Region)
	var deletedIds []*string

	for _, id := range ids {
		param := &ec2.DeleteTransitGatewayVpcAttachmentInput{
			TransitGatewayAttachmentId: id,
		}

		_, err := client.DeleteTransitGatewayVpcAttachment(ctx, param)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: resourceType,
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Transit Gateway Vpc Attachment: %s", *id)
		}
	}

	if waiterr := waitForTransitGatewayAttachmentsToBeDeleted(ctx, client, aws.ToStringSlice(ids)); waiterr != nil {
		return errors.WithStackTrace(waiterr)
	}
	logging.Debugf("[OK] %d Transit Gateway Vpc Attachment(s) deleted in %s", len(deletedIds), scope.Region)
	return nil
}

// waitForTransitGatewayAttachmentsToBeDeleted waits for all Transit Gateway attachments to be deleted.
func waitForTransitGatewayAttachmentsToBeDeleted(ctx context.Context, client TransitGatewaysVpcAttachmentAPI, ids []string) error {
	for i := 0; i < 30; i++ {
		gateways, err := client.DescribeTransitGatewayVpcAttachments(
			ctx, &ec2.DescribeTransitGatewayVpcAttachmentsInput{
				TransitGatewayAttachmentIds: ids,
				Filters: []types.Filter{
					{
						Name:   aws.String("state"),
						Values: []string{"deleting"},
					},
				},
			},
		)
		if err != nil {
			return err
		}
		if len(gateways.TransitGatewayVpcAttachments) == 0 {
			return nil
		}
		logging.Info("Waiting for transit gateways attachments to be deleted...")
		time.Sleep(10 * time.Second)
	}

	return goerror.New("timed out waiting for transit gateway attachments to be successfully deleted")
}

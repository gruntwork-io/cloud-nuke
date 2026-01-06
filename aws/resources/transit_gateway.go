package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// TransitGatewaysAPI defines the interface for Transit Gateway operations.
type TransitGatewaysAPI interface {
	DescribeTransitGateways(ctx context.Context, params *ec2.DescribeTransitGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewaysOutput, error)
	DeleteTransitGateway(ctx context.Context, params *ec2.DeleteTransitGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayOutput, error)
	DescribeTransitGatewayAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayAttachmentsOutput, error)
	DeleteTransitGatewayPeeringAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayPeeringAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error)
	DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error)
	DeleteTransitGatewayConnect(ctx context.Context, params *ec2.DeleteTransitGatewayConnectInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayConnectOutput, error)
}

// NewTransitGateways creates a new TransitGateways resource using the generic resource pattern.
func NewTransitGateways() AwsResource {
	return NewAwsResource(&resource.Resource[TransitGatewaysAPI]{
		ResourceTypeName: "transit-gateway",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[TransitGatewaysAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.TransitGateway
		},
		Lister:             listTransitGateways,
		Nuker:              resource.SequentialDeleter(nukeTransitGateway),
		PermissionVerifier: verifyTransitGatewayNukePermission,
	})
}

// listTransitGateways returns a formatted string of TransitGateway IDs.
// Uses pagination to handle large numbers of transit gateways.
func listTransitGateways(ctx context.Context, client TransitGatewaysAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string
	currentOwner := ctx.Value(util.AccountIdKey)

	paginator := ec2.NewDescribeTransitGatewaysPaginator(client, &ec2.DescribeTransitGatewaysInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[DescribeTransitGateways Failed] %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, transitGateway := range page.TransitGateways {
			// Skip deleted/deleting transit gateways
			if transitGateway.State == types.TransitGatewayStateDeleted || transitGateway.State == types.TransitGatewayStateDeleting {
				continue
			}

			// Skip if owned by a different account
			if currentOwner != nil && transitGateway.OwnerId != nil && currentOwner != aws.ToString(transitGateway.OwnerId) {
				logging.Debugf("[Skipping] Transit Gateway %s owned by different account", aws.ToString(transitGateway.TransitGatewayId))
				continue
			}

			hostNameTagValue := util.GetEC2ResourceNameTagValue(transitGateway.Tags)
			if !cfg.ShouldInclude(config.ResourceValue{
				Time: transitGateway.CreationTime,
				Name: hostNameTagValue,
			}) {
				continue
			}

			ids = append(ids, transitGateway.TransitGatewayId)
		}
	}

	return ids, nil
}

// verifyTransitGatewayNukePermission performs a dry-run delete to check permissions.
// Returns nil if the user has permission, otherwise returns an error.
func verifyTransitGatewayNukePermission(ctx context.Context, client TransitGatewaysAPI, id *string) error {
	params := &ec2.DeleteTransitGatewayInput{
		TransitGatewayId: id,
		DryRun:           aws.Bool(true), // dry run set as true, checks permission without actually making the request
	}
	_, err := client.DeleteTransitGateway(ctx, params)
	if err != nil {
		return util.TransformAWSError(err)
	}
	return nil
}

func nukeTransitGateway(ctx context.Context, client TransitGatewaysAPI, id *string) error {
	// check the transit gateway has attachments and nuke them before
	if err := nukeTransitGatewayAttachments(ctx, client, id); err != nil {
		return errors.WithStackTrace(err)
	}

	if _, err := client.DeleteTransitGateway(ctx, &ec2.DeleteTransitGatewayInput{
		TransitGatewayId: id,
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func nukeTransitGatewayAttachments(ctx context.Context, client TransitGatewaysAPI, id *string) error {
	logging.Debugf("Deleting transit gateway attachments for %s", aws.ToString(id))

	// Use pagination to get all attachments
	paginator := ec2.NewDescribeTransitGatewayAttachmentsPaginator(client, &ec2.DescribeTransitGatewayAttachmentsInput{
		Filters: []types.Filter{
			{Name: aws.String("transit-gateway-id"), Values: []string{aws.ToString(id)}},
			{Name: aws.String("state"), Values: []string{"available"}},
		},
	})

	var attachments []types.TransitGatewayAttachment
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}
		attachments = append(attachments, page.TransitGatewayAttachments...)
	}

	logging.Debugf("Found %d attachment(s) for transit gateway %s", len(attachments), aws.ToString(id))

	for _, attachment := range attachments {
		if err := deleteTransitGatewayAttachment(ctx, client, &attachment); err != nil {
			return err
		}
	}

	// Wait for all attachments to be deleted
	if len(attachments) > 0 {
		if err := waitUntilTransitGatewayAttachmentsDeleted(ctx, client, id); err != nil {
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("Successfully deleted all attachments for transit gateway %s", aws.ToString(id))
	return nil
}

// deleteTransitGatewayAttachment deletes a single transit gateway attachment based on its type.
func deleteTransitGatewayAttachment(ctx context.Context, client TransitGatewaysAPI, attachment *types.TransitGatewayAttachment) error {
	attachmentID := aws.ToString(attachment.TransitGatewayAttachmentId)
	resourceType := attachment.ResourceType

	logging.Debugf("Deleting %s attachment %s", resourceType, attachmentID)

	var err error
	switch resourceType {
	case types.TransitGatewayAttachmentResourceTypePeering:
		_, err = client.DeleteTransitGatewayPeeringAttachment(ctx, &ec2.DeleteTransitGatewayPeeringAttachmentInput{
			TransitGatewayAttachmentId: attachment.TransitGatewayAttachmentId,
		})
	case types.TransitGatewayAttachmentResourceTypeVpc:
		_, err = client.DeleteTransitGatewayVpcAttachment(ctx, &ec2.DeleteTransitGatewayVpcAttachmentInput{
			TransitGatewayAttachmentId: attachment.TransitGatewayAttachmentId,
		})
	case types.TransitGatewayAttachmentResourceTypeConnect:
		_, err = client.DeleteTransitGatewayConnect(ctx, &ec2.DeleteTransitGatewayConnectInput{
			TransitGatewayAttachmentId: attachment.TransitGatewayAttachmentId,
		})
	default:
		return fmt.Errorf("unsupported transit gateway attachment type: %s", resourceType)
	}

	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

// waitUntilTransitGatewayAttachmentsDeleted waits for all attachments on a transit gateway to be deleted.
func waitUntilTransitGatewayAttachmentsDeleted(ctx context.Context, client TransitGatewaysAPI, id *string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timed out waiting for transit gateway attachments to be deleted")
		case <-ticker.C:
			output, err := client.DescribeTransitGatewayAttachments(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{
				Filters: []types.Filter{
					{Name: aws.String("transit-gateway-id"), Values: []string{aws.ToString(id)}},
					{Name: aws.String("state"), Values: []string{"available", "deleting"}},
				},
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}

			if len(output.TransitGatewayAttachments) == 0 {
				return nil
			}
			logging.Debugf("Waiting for %d attachment(s) to be deleted...", len(output.TransitGatewayAttachments))
		}
	}
}

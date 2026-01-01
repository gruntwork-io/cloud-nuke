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
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	TransitGatewayAttachmentTypePeering = "peering"
	TransitGatewayAttachmentTypeVPC     = "vpc"
	TransitGatewayAttachmentTypeConnect = "connect"
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

// transitGatewayState holds shared state for permission verification
type transitGatewayState struct {
	nukables map[string]error
}

var globalTransitGatewayState = &transitGatewayState{
	nukables: make(map[string]error),
}

// NewTransitGateways creates a new TransitGateways resource using the generic resource pattern.
func NewTransitGateways() AwsResource {
	return NewAwsResource(&resource.Resource[TransitGatewaysAPI]{
		ResourceTypeName: "transit-gateway",
		BatchSize:        maxBatchSize,
		InitClient: func(r *resource.Resource[TransitGatewaysAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EC2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ec2.NewFromConfig(awsCfg)
			// Reset global state on init
			globalTransitGatewayState.nukables = make(map[string]error)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.TransitGateway
		},
		Lister:             listTransitGateways,
		Nuker:              deleteTransitGateways,
		PermissionVerifier: verifyTransitGatewayPermission,
	})
}

// [Note 1] :  NOTE on the Approach used:-Using the `dry run` approach on verifying the nuking permission in case of a scoped IAM role.
// IAM:simulateCustomPolicy : could also be used but the IAM role itself needs permission for simulateCustomPolicy method
// else this would not get the desired result. Also in case of multiple t-gateway, if only some has permission to be nuked,
// the t-gateway resource ids needs to be passed individually inside the IAM:simulateCustomPolicy to get the desired result,
// else all would result in `Implicit-deny` as response- this might increase the time complexity. Using dry run to avoid this.

// listTransitGateways returns a formatted string of TransitGateway IDs
func listTransitGateways(ctx context.Context, client TransitGatewaysAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeTransitGateways(ctx, &ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		logging.Debugf("[DescribeTransitGateways Failed] %s", err)
		return nil, errors.WithStackTrace(err)
	}

	currentOwner := ctx.Value(util.AccountIdKey)
	var ids []*string
	for _, transitGateway := range result.TransitGateways {
		hostNameTagValue := util.GetEC2ResourceNameTagValue(transitGateway.Tags)

		if cfg.ShouldInclude(config.ResourceValue{
			Time: transitGateway.CreationTime,
			Name: hostNameTagValue,
		}) &&
			transitGateway.State != types.TransitGatewayStateDeleted && transitGateway.State != types.TransitGatewayStateDeleting {
			ids = append(ids, transitGateway.TransitGatewayId)
		}

		if currentOwner != nil && transitGateway.OwnerId != nil && currentOwner != aws.ToString(transitGateway.OwnerId) {
			globalTransitGatewayState.nukables[*transitGateway.TransitGatewayId] = util.ErrDifferentOwner
			continue
		}
	}

	return ids, nil
}

// verifyTransitGatewayPermission performs a dry-run delete to check permissions.
func verifyTransitGatewayPermission(ctx context.Context, client TransitGatewaysAPI, id *string) error {
	params := &ec2.DeleteTransitGatewayInput{
		TransitGatewayId: id,
		DryRun:           aws.Bool(true), // dry run set as true, checks permission without actually making the request
	}
	_, err := client.DeleteTransitGateway(ctx, params)

	// Store result in global state for use during nuke
	if err != nil {
		globalTransitGatewayState.nukables[*id] = util.TransformAWSError(err)
	} else {
		globalTransitGatewayState.nukables[*id] = nil
	}

	return err
}

// deleteTransitGateways deletes all TransitGateways
// it attempts to nuke only those resources for which the current IAM user has permission
func deleteTransitGateways(ctx context.Context, client TransitGatewaysAPI, scope resource.Scope, resourceType string, ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Transit Gateways to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting all Transit Gateways in region %s", scope.Region)
	var deletedIds []*string

	for _, id := range ids {
		// Check the id has the permission to nuke, if not, continue the execution
		if err, ok := globalTransitGatewayState.nukables[*id]; ok && err != nil {
			// Not adding the report on final result hence not adding a record entry here
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}
		err := nukeTransitGateway(ctx, client, id)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: "Transit Gateway",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Transit Gateway: %s", *id)
		}
	}

	logging.Debugf("[OK] %d Transit Gateway(s) deleted in %s", len(deletedIds), scope.Region)
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
	logging.Debugf("nuking transit gateway attachments for %v", aws.ToString(id))
	output, err := client.DescribeTransitGatewayAttachments(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{
		Filters: []types.Filter{
			{
				Name: aws.String("transit-gateway-id"),
				Values: []string{
					aws.ToString(id),
				},
			},
			{
				Name: aws.String("state"),
				Values: []string{
					"available",
				},
			},
		},
	})
	if err != nil {
		logging.Errorf("[Failed] unable to describe the transit gateway attachments for %v : %s", aws.ToString(id), err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("%v attachment(s) found with %v", len(output.TransitGatewayAttachments), aws.ToString(id))

	for _, attachments := range output.TransitGatewayAttachments {
		var (
			attachmentErr  error
			attachmentType = attachments.ResourceType
			now            = time.Now()
		)

		switch attachmentType {
		case TransitGatewayAttachmentTypePeering:
			logging.Debugf("[Execution] deleting the attachments of type %v for %v ", attachmentType, aws.ToString(id))
			// Delete the Transit Gateway Peering Attachment
			_, attachmentErr = client.DeleteTransitGatewayPeeringAttachment(ctx, &ec2.DeleteTransitGatewayPeeringAttachmentInput{
				TransitGatewayAttachmentId: attachments.TransitGatewayAttachmentId,
			})
		case TransitGatewayAttachmentTypeVPC:
			logging.Debugf("[Execution] deleting the attachments of type %v for %v ", attachmentType, aws.ToString(id))
			// Delete the Transit Gateway VPC Attachment
			_, attachmentErr = client.DeleteTransitGatewayVpcAttachment(ctx, &ec2.DeleteTransitGatewayVpcAttachmentInput{
				TransitGatewayAttachmentId: attachments.TransitGatewayAttachmentId,
			})
		case TransitGatewayAttachmentTypeConnect:
			logging.Debugf("[Execution] deleting the attachments of type %v for %v ", attachmentType, aws.ToString(id))
			// Delete the Transit Gateway Connect Attachment
			_, attachmentErr = client.DeleteTransitGatewayConnect(ctx, &ec2.DeleteTransitGatewayConnectInput{
				TransitGatewayAttachmentId: attachments.TransitGatewayAttachmentId,
			})
		default:
			attachmentErr = fmt.Errorf("%v typed transit gateway attachment nuking not handled", attachmentType)
		}
		if attachmentErr != nil {
			logging.Errorf("[Failed] unable to delete the transit gateway peering attachment for %v : %s", aws.ToString(id), attachmentErr)
			return attachmentErr
		}
		if err := waitUntilTransitGatewayAttachmentDeleted(ctx, client, id, attachmentType); err != nil {
			logging.Errorf("[Failed] unable to wait until nuking the transit gateway attachment with type %v for %v : %s", attachmentType, aws.ToString(id), err)
			return errors.WithStackTrace(err)
		}

		logging.Debugf("waited %v to nuke the attachment", time.Since(now))
	}

	logging.Debugf("[Ok] successfully nuked all the attachments on %v", aws.ToString(id))
	return nil
}

func waitUntilTransitGatewayAttachmentDeleted(ctx context.Context, client TransitGatewaysAPI, id *string, attachmentType types.TransitGatewayAttachmentResourceType) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("transit gateway attachments deletion check timed out after 5 minute")
		case <-ticker.C:
			output, err := client.DescribeTransitGatewayAttachments(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{
				Filters: []types.Filter{
					{
						Name: aws.String("transit-gateway-id"),
						Values: []string{
							aws.ToString(id),
						},
					},
					{
						Name: aws.String("state"),
						Values: []string{
							"available",
							"deleting",
						},
					},
				},
			})
			if err != nil {
				logging.Debugf("transit gateway attachment(s) as type %v existence checking error : %v", attachmentType, err)
				return errors.WithStackTrace(err)
			}

			if len(output.TransitGatewayAttachments) == 0 {
				return nil
			}
			logging.Debugf("%v transit gateway attachments(s) still exists as type %v, waiting...", len(output.TransitGatewayAttachments), attachmentType)
		}
	}
}

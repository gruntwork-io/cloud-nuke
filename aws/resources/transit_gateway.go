package resources

import (
	"context"
	"fmt"
	"time"

	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	TransitGatewayAttachmentTypePeering = "peering"
	TransitGatewayAttachmentTypeVPC     = "vpc"
	TransitGatewayAttachmentTypeConnect = "connect"
)

// [Note 1] :  NOTE on the Apporach used:-Using the `dry run` approach on verifying the nuking permission in case of a scoped IAM role.
// IAM:simulateCustomPolicy : could also be used but the IAM role itself needs permission for simulateCustomPolicy method
//else this would not get the desired result. Also in case of multiple t-gateway, if only some has permssion to be nuked,
// the t-gateway resource ids needs to be passed individually inside the IAM:simulateCustomPolicy to get the desired result,
// else all would result in `Implicit-deny` as response- this might increase the time complexity.Using dry run to avoid this.

// Returns a formatted string of TransitGateway IDs
func (tgw *TransitGateways) getAll(c context.Context, configObj config.Config) ([]*string, error) {

	result, err := tgw.Client.DescribeTransitGateways(tgw.Context, &ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		logging.Debugf("[DescribeTransitGateways Failed] %s", err)
		return nil, errors.WithStackTrace(err)
	}

	currentOwner := c.Value(util.AccountIdKey)
	var ids []*string
	for _, transitGateway := range result.TransitGateways {
		hostNameTagValue := util.GetEC2ResourceNameTagValue(transitGateway.Tags)

		if configObj.TransitGateway.ShouldInclude(config.ResourceValue{
			Time: transitGateway.CreationTime,
			Name: hostNameTagValue,
		}) &&
			transitGateway.State != types.TransitGatewayStateDeleted && transitGateway.State != types.TransitGatewayStateDeleting {
			ids = append(ids, transitGateway.TransitGatewayId)
		}

		if currentOwner != nil && transitGateway.OwnerId != nil && currentOwner != awsgo.ToString(transitGateway.OwnerId) {
			tgw.SetNukableStatus(*transitGateway.TransitGatewayId, util.ErrDifferentOwner)
			continue
		}
	}

	// Check and verfiy the list of allowed nuke actions
	// VerifyNukablePermissions is used to iterate over a list of Transit Gateway IDs (ids) and execute a provided function (func(id *string) error).
	// The function, attempts to delete a Transit Gateway with the specified ID in a dry-run mode (checking permissions without actually performing the delete operation). The result of this operation (error or success) is then captured.
	// See more at [Note 1]
	tgw.VerifyNukablePermissions(ids, func(id *string) error {
		params := &ec2.DeleteTransitGatewayInput{
			TransitGatewayId: id,
			DryRun:           awsgo.Bool(true), // dry run set as true , checks permission without actualy making the request
		}
		_, err := tgw.Client.DeleteTransitGateway(tgw.Context, params)
		return err
	})

	return ids, nil
}

func (tgw *TransitGateways) nuke(id *string) error {
	// check the transit gateway has attachments and nuke them before
	if err := tgw.nukeAttachments(id); err != nil {
		return errors.WithStackTrace(err)
	}

	if _, err := tgw.Client.DeleteTransitGateway(tgw.Context, &ec2.DeleteTransitGatewayInput{
		TransitGatewayId: id,
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (tgw *TransitGateways) nukeAttachments(id *string) error {
	logging.Debugf("nuking transit gateway attachments for %v", awsgo.ToString(id))
	output, err := tgw.Client.DescribeTransitGatewayAttachments(tgw.Context, &ec2.DescribeTransitGatewayAttachmentsInput{
		Filters: []types.Filter{
			{
				Name: awsgo.String("transit-gateway-id"),
				Values: []string{
					awsgo.ToString(id),
				},
			},
			{
				Name: awsgo.String("state"),
				Values: []string{
					"available",
				},
			},
		},
	})
	if err != nil {
		logging.Errorf("[Failed] unable to describe the  transit gateway attachments for %v : %s", awsgo.ToString(id), err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("%v attachment(s) found with %v", len(output.TransitGatewayAttachments), awsgo.ToString(id))

	for _, attachments := range output.TransitGatewayAttachments {
		var (
			err            error
			attachmentType = attachments.ResourceType
			now            = time.Now()
		)

		switch attachmentType {
		case TransitGatewayAttachmentTypePeering:
			logging.Debugf("[Execution] deleting the attachments of type %v for %v ", attachmentType, awsgo.ToString(id))
			// Delete the Transit Gateway Peering Attachment
			_, err = tgw.Client.DeleteTransitGatewayPeeringAttachment(tgw.Context, &ec2.DeleteTransitGatewayPeeringAttachmentInput{
				TransitGatewayAttachmentId: attachments.TransitGatewayAttachmentId,
			})
		case TransitGatewayAttachmentTypeVPC:
			logging.Debugf("[Execution] deleting the attachments of type %v for %v ", attachmentType, awsgo.ToString(id))
			// Delete the Transit Gateway VPC Attachment
			_, err = tgw.Client.DeleteTransitGatewayVpcAttachment(tgw.Context, &ec2.DeleteTransitGatewayVpcAttachmentInput{
				TransitGatewayAttachmentId: attachments.TransitGatewayAttachmentId,
			})
		case TransitGatewayAttachmentTypeConnect:
			logging.Debugf("[Execution] deleting the attachments of type %v for %v ", attachmentType, awsgo.ToString(id))
			// Delete the Transit Gateway Connect Attachment
			_, err = tgw.Client.DeleteTransitGatewayConnect(tgw.Context, &ec2.DeleteTransitGatewayConnectInput{
				TransitGatewayAttachmentId: attachments.TransitGatewayAttachmentId,
			})
		default:
			err = fmt.Errorf("%v typed transit gateway attachment nuking not handled", attachmentType)
		}
		if err != nil {
			logging.Errorf("[Failed] unable to delete the  transit gateway peernig attachment for %v : %s", awsgo.ToString(id), err)
			return err
		}
		if err := tgw.WaitUntilTransitGatewayAttachmentDeleted(id, attachmentType); err != nil {
			logging.Errorf("[Failed] unable to wait until nuking the transit gateway attachment with type %v for %v : %s", attachmentType, awsgo.ToString(id), err)
			return errors.WithStackTrace(err)
		}

		logging.Debugf("waited %v to nuke the attachment", time.Since(now))
	}

	logging.Debugf("[Ok] successfully nuked all the attachments on %v", awsgo.ToString(id))
	return nil
}

func (tgw *TransitGateways) WaitUntilTransitGatewayAttachmentDeleted(id *string, attachmentType types.TransitGatewayAttachmentResourceType) error {
	timeoutCtx, cancel := context.WithTimeout(tgw.Context, 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("transit gateway attachments deletion check timed out after 5 minute")
		case <-ticker.C:
			output, err := tgw.Client.DescribeTransitGatewayAttachments(tgw.Context, &ec2.DescribeTransitGatewayAttachmentsInput{
				Filters: []types.Filter{
					{
						Name: awsgo.String("transit-gateway-id"),
						Values: []string{
							awsgo.ToString(id),
						},
					},
					{
						Name: awsgo.String("state"),
						Values: []string{
							"available",
							"deleting",
						},
					},
				},
			})
			if err != nil {
				logging.Debugf("transit gateway attachment(s) as type %v existance checking error : %v", attachmentType, err)
				return errors.WithStackTrace(err)
			}

			if len(output.TransitGatewayAttachments) == 0 {
				return nil
			}
			logging.Debugf("%v transit gateway attachments(s) still exists as type %v, waiting...", len(output.TransitGatewayAttachments), attachmentType)
		}
	}
}

// Delete all TransitGateways
// it attempts to nuke only those resources for which the current IAM user has permission
func (tgw *TransitGateways) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Transit Gateways to nuke in region %s", tgw.Region)
		return nil
	}

	logging.Debugf("Deleting all Transit Gateways in region %s", tgw.Region)
	var deletedIds []*string

	for _, id := range ids {
		//check the id has the permission to nuke, if not. continue the execution
		if nukable, reason := tgw.IsNukable(*id); !nukable {
			//not adding the report on final result hence not adding a record entry here
			logging.Debugf("[Skipping] %s nuke because %v", *id, reason)
			continue
		}
		err := tgw.nuke(id)

		// Record status of this resource
		e := report.Entry{
			Identifier:   awsgo.ToString(id),
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

	logging.Debugf("[OK] %d Transit Gateway(s) deleted in %s", len(deletedIds), tgw.Region)
	return nil
}

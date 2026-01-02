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
)

// InternetGatewayAPI defines the interface for Internet Gateway operations.
type InternetGatewayAPI interface {
	DescribeInternetGateways(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DeleteInternetGateway(ctx context.Context, params *ec2.DeleteInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteInternetGatewayOutput, error)
	DetachInternetGateway(ctx context.Context, params *ec2.DetachInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DetachInternetGatewayOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// NewInternetGateway creates a new InternetGateway resource using the generic resource pattern.
func NewInternetGateway() AwsResource {
	return NewEC2AwsResource[InternetGatewayAPI](
		"internet-gateway",
		WrapAwsInitClient(func(r *resource.Resource[InternetGatewayAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		func(c config.Config) config.EC2ResourceType { return c.InternetGateway },
		listInternetGateways,
		resource.MultiStepDeleter(detachInternetGateway, deleteInternetGateway),
		&EC2ResourceOptions[InternetGatewayAPI]{PermissionVerifier: verifyInternetGatewayPermission},
	)
}

// verifyInternetGatewayPermission performs a dry-run delete to check permissions.
func verifyInternetGatewayPermission(ctx context.Context, client InternetGatewayAPI, id *string) error {
	_, err := client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: id,
		DryRun:            aws.Bool(true),
	})
	return err
}

// listInternetGateways retrieves all Internet Gateways that match the config filters.
// When defaultOnly is true, only IGWs attached to default VPCs are returned (for defaults-aws command).
func listInternetGateways(ctx context.Context, client InternetGatewayAPI, scope resource.Scope, cfg config.ResourceType, defaultOnly bool) ([]*string, error) {
	// When defaultOnly is true, get the list of default VPC IDs to filter by
	var defaultVpcIds map[string]bool
	if defaultOnly {
		defaultVpcIds = make(map[string]bool)
		vpcs, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			Filters: []types.Filter{
				{Name: aws.String("is-default"), Values: []string{"true"}},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe default VPCs: %w", err)
		}
		for _, vpc := range vpcs.Vpcs {
			defaultVpcIds[aws.ToString(vpc.VpcId)] = true
		}
		if len(defaultVpcIds) == 0 {
			logging.Debugf("[Internet Gateway] No default VPCs found, skipping")
			return nil, nil
		}
	}

	var identifiers []*string
	paginator := ec2.NewDescribeInternetGatewaysPaginator(client, &ec2.DescribeInternetGatewaysInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Internet Gateway] Failed to list internet gateways: %s", err)
			return nil, err
		}

		for _, ig := range page.InternetGateways {
			// When defaultOnly is true, skip IGWs not attached to default VPCs
			if defaultOnly {
				attachedToDefault := false
				for _, att := range ig.Attachments {
					if defaultVpcIds[aws.ToString(att.VpcId)] {
						attachedToDefault = true
						break
					}
				}
				if !attachedToDefault {
					continue
				}
			}

			tagMap := util.ConvertTypesTagsToMap(ig.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, ig.InternetGatewayId, tagMap)
			if err != nil {
				logging.Errorf("[Internet Gateway] Unable to retrieve first seen tag for %s: %s", aws.ToString(ig.InternetGatewayId), err)
				return nil, err
			}

			if shouldIncludeInternetGateway(ig, firstSeenTime, cfg) {
				identifiers = append(identifiers, ig.InternetGatewayId)
			}
		}
	}

	return identifiers, nil
}

// shouldIncludeInternetGateway determines if an internet gateway should be included based on config filters.
func shouldIncludeInternetGateway(ig types.InternetGateway, firstSeenTime *time.Time, cfg config.ResourceType) bool {
	tagMap := util.ConvertTypesTagsToMap(ig.Tags)
	var name string
	if n, ok := tagMap["Name"]; ok {
		name = n
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: &name,
		Tags: tagMap,
		Time: firstSeenTime,
	})
}

// detachInternetGateway detaches an internet gateway from its VPC.
func detachInternetGateway(ctx context.Context, client InternetGatewayAPI, id *string) error {
	igID := aws.ToString(id)
	logging.Debugf("[Internet Gateway] Looking up VPC attachment for: %s", igID)

	// Query AWS to get the current VPC attachment
	resp, err := client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []string{igID},
	})
	if err != nil {
		return fmt.Errorf("failed to describe internet gateway %s: %w", igID, err)
	}

	if len(resp.InternetGateways) == 0 {
		logging.Debugf("[Internet Gateway] Not found: %s", igID)
		return nil
	}

	ig := resp.InternetGateways[0]
	if len(ig.Attachments) == 0 {
		logging.Debugf("[Internet Gateway] No VPC attachment found for %s, skipping detach", igID)
		return nil
	}

	vpcID := aws.ToString(ig.Attachments[0].VpcId)
	logging.Debugf("[Internet Gateway] Detaching %s from VPC %s", igID, vpcID)

	_, err = client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
		InternetGatewayId: id,
		VpcId:             aws.String(vpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to detach internet gateway %s from VPC %s: %w", igID, vpcID, err)
	}

	logging.Debugf("[Internet Gateway] Successfully detached %s from VPC %s", igID, vpcID)
	return nil
}

// deleteInternetGateway deletes an internet gateway.
func deleteInternetGateway(ctx context.Context, client InternetGatewayAPI, id *string) error {
	logging.Debugf("Deleting Internet Gateway %s", aws.ToString(id))
	_, err := client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: id,
	})
	if err != nil {
		return fmt.Errorf("failed to delete internet gateway %s: %w", aws.ToString(id), err)
	}

	logging.Debugf("Successfully deleted Internet Gateway %s", aws.ToString(id))
	return nil
}

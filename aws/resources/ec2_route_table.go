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

// RouteTableAPI defines the interface for Route Table operations.
type RouteTableAPI interface {
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	DisassociateRouteTable(ctx context.Context, params *ec2.DisassociateRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateRouteTableOutput, error)
	DeleteRouteTable(ctx context.Context, params *ec2.DeleteRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteTableOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// NewRouteTable creates a new RouteTable resource using the generic resource pattern.
func NewRouteTable() AwsResource {
	return NewEC2AwsResource[RouteTableAPI](
		"route-table",
		WrapAwsInitClient(func(r *resource.Resource[RouteTableAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		func(c config.Config) config.EC2ResourceType { return c.RouteTable },
		listRouteTables,
		resource.MultiStepDeleter(disassociateRouteTableSubnets, deleteRouteTable),
		&EC2ResourceOptions[RouteTableAPI]{PermissionVerifier: verifyRouteTableNukePermission},
	)
}

// verifyRouteTableNukePermission performs a dry-run delete to check permissions.
func verifyRouteTableNukePermission(ctx context.Context, client RouteTableAPI, id *string) error {
	_, err := client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: id,
		DryRun:       aws.Bool(true),
	})
	return util.TransformAWSError(err)
}

// listRouteTables retrieves all non-main route tables that match the config filters.
// Main route tables are automatically deleted when their VPC is deleted.
// When defaultOnly is true, only route tables in default VPCs are returned (for defaults-aws command).
func listRouteTables(ctx context.Context, client RouteTableAPI, scope resource.Scope, cfg config.ResourceType, defaultOnly bool) ([]*string, error) {
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
			logging.Debugf("[Route Table] No default VPCs found, skipping")
			return nil, nil
		}
	}

	var identifiers []*string

	paginator := ec2.NewDescribeRouteTablesPaginator(client, &ec2.DescribeRouteTablesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Route Table] Failed to list route tables: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, rt := range page.RouteTables {
			// Skip the main route table — it is auto-deleted with the VPC
			if isMainRouteTable(rt) {
				continue
			}

			// When defaultOnly is true, skip route tables not in default VPCs
			if defaultOnly && !defaultVpcIds[aws.ToString(rt.VpcId)] {
				continue
			}

			tagMap := util.ConvertTypesTagsToMap(rt.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, rt.RouteTableId, tagMap)
			if err != nil {
				logging.Errorf("[Route Table] Unable to retrieve first seen tag for %s: %s", aws.ToString(rt.RouteTableId), err)
				continue
			}

			if shouldIncludeRouteTable(rt, firstSeenTime, cfg) {
				identifiers = append(identifiers, rt.RouteTableId)
			}
		}
	}

	return identifiers, nil
}

// isMainRouteTable returns true if the route table is the main route table for its VPC.
// Main route tables are auto-deleted when the VPC is deleted and cannot be explicitly deleted.
func isMainRouteTable(rt types.RouteTable) bool {
	for _, assoc := range rt.Associations {
		if assoc.Main != nil && *assoc.Main {
			return true
		}
	}
	return false
}

// shouldIncludeRouteTable determines if a route table should be included for deletion.
func shouldIncludeRouteTable(rt types.RouteTable, firstSeenTime *time.Time, cfg config.ResourceType) bool {
	tagMap := util.ConvertTypesTagsToMap(rt.Tags)
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

// disassociateRouteTableSubnets disassociates all subnet associations from a route table.
// Main associations cannot be disassociated — they are skipped.
func disassociateRouteTableSubnets(ctx context.Context, client RouteTableAPI, id *string) error {
	rtID := aws.ToString(id)
	logging.Debugf("[Route Table] Disassociating subnets for: %s", rtID)

	resp, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		RouteTableIds: []string{rtID},
	})
	if err != nil {
		logging.Debugf("[Route Table] Failed to describe: %s", err)
		return errors.WithStackTrace(err)
	}

	if len(resp.RouteTables) == 0 {
		logging.Debugf("[Route Table] Not found: %s", rtID)
		return nil
	}

	for _, assoc := range resp.RouteTables[0].Associations {
		if assoc.Main != nil && *assoc.Main {
			continue
		}

		assocID := aws.ToString(assoc.RouteTableAssociationId)
		logging.Debugf("[Route Table] Disassociating %s from %s", assocID, rtID)

		_, err := client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
			AssociationId: assoc.RouteTableAssociationId,
		})
		if err != nil {
			logging.Debugf("[Route Table] Failed to disassociate %s: %s", assocID, err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[Route Table] Successfully disassociated all subnets for: %s", rtID)
	return nil
}

// deleteRouteTable deletes a single route table.
func deleteRouteTable(ctx context.Context, client RouteTableAPI, id *string) error {
	logging.Debugf("[Route Table] Deleting: %s", aws.ToString(id))

	_, err := client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: id,
	})
	if err != nil {
		logging.Debugf("[Route Table] Failed to delete %s: %s", aws.ToString(id), err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[Route Table] Successfully deleted: %s", aws.ToString(id))
	return nil
}

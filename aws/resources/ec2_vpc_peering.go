package resources

import (
	"context"
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

// VPCPeeringAPI defines the interface for VPC Peering Connection operations.
type VPCPeeringAPI interface {
	DescribeVpcPeeringConnections(ctx context.Context, params *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error)
	DeleteVpcPeeringConnection(ctx context.Context, params *ec2.DeleteVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcPeeringConnectionOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// NewVPCPeeringConnection creates a new VPC Peering Connection resource.
func NewVPCPeeringConnection() AwsResource {
	return NewAwsResource(&resource.Resource[VPCPeeringAPI]{
		ResourceTypeName: "vpc-peering-connection",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[VPCPeeringAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.VPCPeeringConnection
		},
		Lister:             listVPCPeeringConnections,
		Nuker:              resource.SimpleBatchDeleter(deleteVpcPeeringConnection),
		PermissionVerifier: verifyVPCPeeringNukePermission,
	})
}

// verifyVPCPeeringNukePermission performs a dry-run delete to check permissions.
func verifyVPCPeeringNukePermission(ctx context.Context, client VPCPeeringAPI, id *string) error {
	_, err := client.DeleteVpcPeeringConnection(ctx, &ec2.DeleteVpcPeeringConnectionInput{
		VpcPeeringConnectionId: id,
		DryRun:                 aws.Bool(true),
	})
	return util.TransformAWSError(err)
}

// terminalPeeringStates contains the VPC peering connection states that should be excluded from listing.
var terminalPeeringStates = map[types.VpcPeeringConnectionStateReasonCode]bool{
	types.VpcPeeringConnectionStateReasonCodeDeleted:  true,
	types.VpcPeeringConnectionStateReasonCodeDeleting: true,
	types.VpcPeeringConnectionStateReasonCodeRejected: true,
	types.VpcPeeringConnectionStateReasonCodeFailed:   true,
	types.VpcPeeringConnectionStateReasonCodeExpired:  true,
}

// listVPCPeeringConnections retrieves all active VPC peering connections that match config filters.
func listVPCPeeringConnections(ctx context.Context, client VPCPeeringAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &ec2.DescribeVpcPeeringConnectionsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[VPC Peering] Failed to list peering connections: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, pcx := range page.VpcPeeringConnections {
			if pcx.Status != nil && terminalPeeringStates[pcx.Status.Code] {
				continue
			}

			tagMap := util.ConvertTypesTagsToMap(pcx.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, pcx.VpcPeeringConnectionId, tagMap)
			if err != nil {
				logging.Errorf("[VPC Peering] Unable to retrieve first seen tag for %s: %s", aws.ToString(pcx.VpcPeeringConnectionId), err)
				continue
			}

			if shouldIncludeVPCPeering(pcx, firstSeenTime, cfg) {
				identifiers = append(identifiers, pcx.VpcPeeringConnectionId)
			}
		}
	}

	return identifiers, nil
}

// shouldIncludeVPCPeering determines if a VPC peering connection should be included for deletion.
func shouldIncludeVPCPeering(pcx types.VpcPeeringConnection, firstSeenTime *time.Time, cfg config.ResourceType) bool {
	tagMap := util.ConvertTypesTagsToMap(pcx.Tags)
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

// deleteVpcPeeringConnection deletes a single VPC peering connection.
func deleteVpcPeeringConnection(ctx context.Context, client VPCPeeringAPI, id *string) error {
	logging.Debugf("[VPC Peering] Deleting: %s", aws.ToString(id))

	_, err := client.DeleteVpcPeeringConnection(ctx, &ec2.DeleteVpcPeeringConnectionInput{
		VpcPeeringConnectionId: id,
	})
	if err != nil {
		logging.Debugf("[VPC Peering] Failed to delete %s: %s", aws.ToString(id), err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[VPC Peering] Successfully deleted: %s", aws.ToString(id))
	return nil
}

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
	DeleteInternetGateway(ctx context.Context, params *ec2.DeleteInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteInternetGatewayOutput, error)
	DetachInternetGateway(ctx context.Context, params *ec2.DetachInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DetachInternetGatewayOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// NewInternetGateway creates a new InternetGateway resource using the generic resource pattern.
func NewInternetGateway() AwsResource {
	return NewAwsResource(&resource.Resource[InternetGatewayAPI]{
		ResourceTypeName: "internet-gateway",
		BatchSize:        50,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[InternetGatewayAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.InternetGateway
		},
		Lister: listInternetGateways,
		Nuker:  resource.MultiStepDeleter(detachInternetGateway, deleteInternetGateway),
		PermissionVerifier: func(ctx context.Context, client InternetGatewayAPI, id *string) error {
			_, err := client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
				InternetGatewayId: id,
				DryRun:            aws.Bool(true),
			})
			return err
		},
	})
}

// listInternetGateways retrieves all Internet Gateways that match the config filters.
func listInternetGateways(ctx context.Context, client InternetGatewayAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := ec2.NewDescribeInternetGatewaysPaginator(client, &ec2.DescribeInternetGatewaysInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Internet Gateway] Failed to list internet gateways: %s", err)
			return nil, err
		}

		for _, ig := range page.InternetGateways {
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

// nukeInternetGateway detaches and deletes an internet gateway with an explicit VPC ID.
// This function is exported for use by ec2_vpc.go when cleaning up VPC resources.
func nukeInternetGateway(client InternetGatewayAPI, gatewayId *string, vpcID string) error {
	ctx := context.Background()

	logging.Debugf("Detaching Internet Gateway %s from VPC %s", aws.ToString(gatewayId), vpcID)
	_, err := client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
		InternetGatewayId: gatewayId,
		VpcId:             aws.String(vpcID),
	})
	if err != nil {
		logging.Debugf("Failed to detach internet gateway %s: %s", aws.ToString(gatewayId), err)
		return err
	}
	logging.Debugf("Successfully detached internet gateway %s", aws.ToString(gatewayId))

	logging.Debugf("Deleting internet gateway %s", aws.ToString(gatewayId))
	_, err = client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: gatewayId,
	})
	if err != nil {
		logging.Debugf("Failed to delete internet gateway %s: %s", aws.ToString(gatewayId), err)
		return err
	}
	logging.Debugf("Successfully deleted internet gateway %s", aws.ToString(gatewayId))

	return nil
}

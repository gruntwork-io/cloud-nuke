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
)

// EC2EndpointsAPI defines the interface for EC2 VPC Endpoints operations.
type EC2EndpointsAPI interface {
	DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error)
	DeleteVpcEndpoints(ctx context.Context, params *ec2.DeleteVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointsOutput, error)
}

// NewEC2Endpoints creates a new EC2 VPC Endpoints resource using the generic resource pattern.
func NewEC2Endpoints() AwsResource {
	return NewAwsResource(&resource.Resource[EC2EndpointsAPI]{
		ResourceTypeName: "ec2-endpoint",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[EC2EndpointsAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EC2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ec2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2Endpoint
		},
		Lister:             listEC2Endpoints,
		Nuker:              resource.SimpleBatchDeleter(deleteEC2Endpoint),
		PermissionVerifier: verifyEC2EndpointPermission,
	})
}

// ShouldIncludeVpcEndpoint determines if a VPC endpoint should be included based on config filters.
func ShouldIncludeVpcEndpoint(endpoint *types.VpcEndpoint, firstSeenTime *time.Time, configObj config.Config) bool {
	var endpointName string
	// get the tags as map
	tagMap := util.ConvertTypesTagsToMap(endpoint.Tags)
	if name, ok := tagMap["Name"]; ok {
		endpointName = name
	}

	return configObj.EC2Endpoint.ShouldInclude(config.ResourceValue{
		Name: &endpointName,
		Time: firstSeenTime,
		Tags: tagMap,
	})
}

// listEC2Endpoints retrieves all VPC endpoints that match the config filters.
func listEC2Endpoints(ctx context.Context, client EC2EndpointsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	endpoints, err := client.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{})
	if err != nil {
		return nil, err
	}

	for _, endpoint := range endpoints.VpcEndpoints {
		firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, endpoint.VpcEndpointId, util.ConvertTypesTagsToMap(endpoint.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, err
		}

		tagMap := util.ConvertTypesTagsToMap(endpoint.Tags)
		var endpointName string
		if name, ok := tagMap["Name"]; ok {
			endpointName = name
		}

		if cfg.ShouldInclude(config.ResourceValue{
			Name: &endpointName,
			Time: firstSeenTime,
			Tags: tagMap,
		}) {
			result = append(result, endpoint.VpcEndpointId)
		}
	}

	return result, nil
}

// deleteEC2Endpoint deletes a single VPC endpoint.
func deleteEC2Endpoint(ctx context.Context, client EC2EndpointsAPI, id *string) error {
	logging.Debugf("Deleting VPC endpoint %s", aws.ToString(id))

	_, err := client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []string{aws.ToString(id)},
	})
	return err
}

// verifyEC2EndpointPermission performs a dry-run delete to check permissions.
func verifyEC2EndpointPermission(ctx context.Context, client EC2EndpointsAPI, id *string) error {
	_, err := client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []string{aws.ToString(id)},
		DryRun:         aws.Bool(true),
	})
	return err
}

// nukeVpcEndpoint deletes VPC endpoints by their IDs.
// This is exported for use by ec2_vpc.go when nuking VPCs.
func nukeVpcEndpoint(client EC2EndpointsAPI, endpointIds []*string) error {
	logging.Debugf("Deleting VPC endpoints %s", aws.ToStringSlice(endpointIds))

	_, err := client.DeleteVpcEndpoints(context.Background(), &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: aws.ToStringSlice(endpointIds),
	})
	if err != nil {
		logging.Debugf("Failed to delete VPC endpoints: %s", err.Error())
		return err
	}

	logging.Debugf("Successfully deleted VPC endpoints %s", aws.ToStringSlice(endpointIds))
	return nil
}

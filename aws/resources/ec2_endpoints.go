package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
		BatchSize:        25, // DeleteVpcEndpoints supports up to 25 endpoints per call
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2EndpointsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2Endpoint
		},
		Lister:             listEC2Endpoints,
		Nuker:              resource.BulkDeleter(deleteEC2Endpoints),
		PermissionVerifier: verifyEC2EndpointPermission,
	})
}

// listEC2Endpoints retrieves all VPC endpoints that match the config filters.
func listEC2Endpoints(ctx context.Context, client EC2EndpointsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	paginator := ec2.NewDescribeVpcEndpointsPaginator(client, &ec2.DescribeVpcEndpointsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, endpoint := range page.VpcEndpoints {
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, endpoint.VpcEndpointId, util.ConvertTypesTagsToMap(endpoint.Tags))
			if err != nil {
				logging.Errorf("Unable to retrieve tags for endpoint %s", aws.ToString(endpoint.VpcEndpointId))
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
	}

	return result, nil
}

// deleteEC2Endpoints deletes VPC endpoints using the bulk delete API.
func deleteEC2Endpoints(ctx context.Context, client EC2EndpointsAPI, ids []string) error {
	logging.Debugf("Deleting VPC endpoints: %v", ids)

	_, err := client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: ids,
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
	if len(endpointIds) == 0 {
		return nil
	}

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

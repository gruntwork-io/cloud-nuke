package resources

import (
	"context"

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
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
}

// NewEC2Endpoints creates a new EC2 VPC Endpoints resource using the generic resource pattern.
func NewEC2Endpoints() AwsResource {
	return NewEC2AwsResource[EC2EndpointsAPI](
		"ec2-endpoint",
		WrapAwsInitClient(func(r *resource.Resource[EC2EndpointsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		func(c config.Config) config.EC2ResourceType { return c.EC2Endpoint },
		listEC2Endpoints,
		resource.BulkDeleter(deleteEC2Endpoints),
		&EC2ResourceOptions[EC2EndpointsAPI]{PermissionVerifier: verifyEC2EndpointPermission},
	)
}

// listEC2Endpoints retrieves all VPC endpoints that match the config filters.
func listEC2Endpoints(ctx context.Context, client EC2EndpointsAPI, scope resource.Scope, cfg config.ResourceType, defaultOnly bool) ([]*string, error) {
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
			return nil, err
		}
		for _, vpc := range vpcs.Vpcs {
			defaultVpcIds[aws.ToString(vpc.VpcId)] = true
		}
		if len(defaultVpcIds) == 0 {
			return nil, nil
		}
	}

	var result []*string

	paginator := ec2.NewDescribeVpcEndpointsPaginator(client, &ec2.DescribeVpcEndpointsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, endpoint := range page.VpcEndpoints {
			// When defaultOnly is true, skip endpoints not in default VPCs
			if defaultOnly && !defaultVpcIds[aws.ToString(endpoint.VpcId)] {
				continue
			}

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

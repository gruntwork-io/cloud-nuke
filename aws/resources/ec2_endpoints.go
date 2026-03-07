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
	"github.com/gruntwork-io/go-commons/retry"
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
		resource.ConcurrentDeleteThenWaitAll(deleteEC2Endpoint, waitForEndpointsDeleted),
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
			// Skip requester-managed endpoints — these are created by AWS services
			// and cannot be deleted directly.
			if aws.ToBool(endpoint.RequesterManaged) {
				logging.Debugf("[Skip] VPC endpoint %s is requester-managed (created by AWS service)", aws.ToString(endpoint.VpcEndpointId))
				continue
			}

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

// deleteEC2Endpoint deletes a single VPC endpoint.
func deleteEC2Endpoint(ctx context.Context, client EC2EndpointsAPI, id *string) error {
	resp, err := client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []string{aws.ToString(id)},
	})
	if err != nil {
		return err
	}
	if resp != nil && len(resp.Unsuccessful) > 0 {
		item := resp.Unsuccessful[0]
		msg := "unknown error"
		if item.Error != nil {
			msg = aws.ToString(item.Error.Message)
		}
		return fmt.Errorf("failed to delete VPC endpoint %s: %s", aws.ToString(id), msg)
	}
	return nil
}

// waitForEndpointsDeleted waits for all VPC Endpoints to finish deleting.
// VPC Endpoint deletion is asynchronous — the API returns immediately but ENIs
// aren't released until the endpoint finishes deleting. Downstream resources like
// subnets will fail with DependencyViolation if we don't wait.
func waitForEndpointsDeleted(ctx context.Context, client EC2EndpointsAPI, ids []string) error {
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	return retry.DoWithRetry(
		logging.Logger.WithTime(time.Now()),
		"Waiting for all VPC Endpoints to be deleted",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			// Query by state filter only — not by ID. AWS removes fully-deleted
			// endpoints from DescribeVpcEndpoints and returns NotFound if any ID
			// in the batch is gone, which would cause false-success when other
			// endpoints are still deleting.
			input := &ec2.DescribeVpcEndpointsInput{
				Filters: []types.Filter{
					{
						Name: aws.String("vpc-endpoint-state"),
						Values: []string{
							"pending",
							"available",
							"deleting",
							"pendingAcceptance",
						},
					},
				},
			}

			remaining := 0
			paginator := ec2.NewDescribeVpcEndpointsPaginator(client, input)
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					return retry.FatalError{Underlying: err}
				}
				for _, ep := range page.VpcEndpoints {
					if idSet[aws.ToString(ep.VpcEndpointId)] {
						remaining++
					}
				}
			}
			if remaining > 0 {
				return fmt.Errorf("%d VPC Endpoints still deleting", remaining)
			}
			return nil
		},
	)
}

// verifyEC2EndpointPermission performs a dry-run delete to check permissions.
func verifyEC2EndpointPermission(ctx context.Context, client EC2EndpointsAPI, id *string) error {
	_, err := client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []string{aws.ToString(id)},
		DryRun:         aws.Bool(true),
	})
	return util.TransformAWSError(err)
}

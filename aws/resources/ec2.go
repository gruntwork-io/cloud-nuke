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
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2InstancesAPI defines the interface for EC2 Instances operations.
type EC2InstancesAPI interface {
	DescribeInstanceAttribute(ctx context.Context, params *ec2.DescribeInstanceAttributeInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceAttributeOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error)
	TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
}

// NewEC2Instances creates a new EC2 Instances resource using the generic resource pattern.
func NewEC2Instances() AwsResource {
	return NewAwsResource(&resource.Resource[EC2InstancesAPI]{
		ResourceTypeName: "ec2",
		// Tentative batch size to ensure AWS doesn't throttle
		BatchSize: 49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2InstancesAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2
		},
		Lister: listEC2Instances,
		Nuker: resource.MultiStepDeleter(
			releaseInstanceEIPs,
			terminateEC2Instance,
			waitForEC2InstanceTerminated,
		),
	})
}

// listEC2Instances retrieves all EC2 instances that match the config filters.
func listEC2Instances(ctx context.Context, client EC2InstancesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	params := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []string{
					"running", "pending",
					"stopped", "stopping",
				},
			},
		},
	}

	var allInstanceIds []*string
	paginator := ec2.NewDescribeInstancesPaginator(client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		instanceIds, err := filterOutProtectedInstances(ctx, client, page, cfg)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		allInstanceIds = append(allInstanceIds, instanceIds...)
	}

	return allInstanceIds, nil
}

// filterOutProtectedInstances returns only instance IDs of unprotected EC2 instances
func filterOutProtectedInstances(ctx context.Context, client EC2InstancesAPI, output *ec2.DescribeInstancesOutput, cfg config.ResourceType) ([]*string, error) {
	var filteredIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId

			attr, err := client.DescribeInstanceAttribute(ctx, &ec2.DescribeInstanceAttributeInput{
				Attribute:  types.InstanceAttributeNameDisableApiTermination,
				InstanceId: aws.String(instanceID),
			})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if shouldIncludeInstanceId(instance, *attr.DisableApiTermination.Value, cfg) {
				filteredIds = append(filteredIds, &instanceID)
			}
		}
	}

	return filteredIds, nil
}

func shouldIncludeInstanceId(instance types.Instance, protected bool, cfg config.ResourceType) bool {
	if protected {
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	instanceName := util.GetEC2ResourceNameTagValue(instance.Tags)
	return cfg.ShouldInclude(config.ResourceValue{
		Name: instanceName,
		Time: instance.LaunchTime,
		Tags: util.ConvertTypesTagsToMap(instance.Tags),
	})
}

// releaseInstanceEIPs releases any Elastic IPs associated with a single EC2 instance.
func releaseInstanceEIPs(ctx context.Context, client EC2InstancesAPI, instanceID *string) error {
	output, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: []string{aws.ToString(instanceID)},
			},
		},
	})
	if err != nil {
		return err
	}

	if output == nil || len(output.Addresses) == 0 {
		return nil
	}

	for _, address := range output.Addresses {
		if address.AllocationId == nil {
			continue
		}

		if _, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: address.AllocationId,
		}); err != nil {
			logging.Debugf("Failed to release EIP %s: %v", aws.ToString(address.AllocationId), err)
			// Continue releasing other EIPs even if one fails
		}
	}

	return nil
}

// terminateEC2Instance terminates a single EC2 instance.
func terminateEC2Instance(ctx context.Context, client EC2InstancesAPI, instanceID *string) error {
	_, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{aws.ToString(instanceID)},
	})
	return err
}

// waitForEC2InstanceTerminated waits for a single EC2 instance to be terminated.
func waitForEC2InstanceTerminated(ctx context.Context, client EC2InstancesAPI, instanceID *string) error {
	waiter := ec2.NewInstanceTerminatedWaiter(client)
	return waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{aws.ToString(instanceID)},
	}, DefaultWaitTimeout)
}

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
		InitClient: func(r *resource.Resource[EC2InstancesAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EC2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ec2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2
		},
		Lister: listEC2Instances,
		Nuker:  deleteEC2Instances,
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

	output, err := client.DescribeInstances(ctx, params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	instanceIds, err := filterOutProtectedInstances(ctx, client, output, cfg)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return instanceIds, nil
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

func releaseEIPs(ctx context.Context, client EC2InstancesAPI, instanceIds []*string) error {
	logging.Debugf("Releasing Elastic IP address(s) associated with instances")

	for _, instanceID := range instanceIds {
		// Get the Elastic IPs associated with the EC2 instances
		output, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
			Filters: []types.Filter{
				{
					Name: aws.String("instance-id"),
					Values: []string{
						aws.ToString(instanceID),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Ensure output is not nil before iterating
		if output == nil || len(output.Addresses) == 0 {
			logging.Debugf("No Elastic IPs found for instance %s", *instanceID)
			continue
		}

		for _, address := range output.Addresses {
			if address.AllocationId == nil {
				logging.Debugf("Skipping Elastic IP release: AllocationId is nil for instance %s", *instanceID)
				continue
			}

			_, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
				AllocationId: address.AllocationId,
			})

			if err != nil {
				logging.Debugf("An error occurred while releasing the Elastic IP address %s, error: %v", *address.AllocationId, err)
				continue
			}

			logging.Debugf("Released Elastic IP address %s from instance %s", *address.AllocationId, *instanceID)
		}
	}

	return nil
}

// deleteEC2Instances deletes all non protected EC2 instances.
func deleteEC2Instances(ctx context.Context, client EC2InstancesAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No EC2 instances to nuke in region %s", scope.Region)
		return nil
	}

	// release the attached elastic ip's
	// Note: This should be done before terminating the EC2 instances
	err := releaseEIPs(ctx, client, identifiers)
	if err != nil {
		logging.Debugf("[Failed EIP release] %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("Terminating all EC2 instances in region %s", scope.Region)

	params := &ec2.TerminateInstancesInput{
		InstanceIds: aws.ToStringSlice(identifiers),
	}

	_, err = client.TerminateInstances(ctx, params)
	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	waiter := ec2.NewInstanceTerminatedWaiter(client)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: aws.ToStringSlice(identifiers),
			},
		},
	}, DefaultWaitTimeout)

	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[OK] %d instance(s) terminated in %s", len(identifiers), scope.Region)
	return nil
}

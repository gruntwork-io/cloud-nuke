package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// returns only instance Ids of unprotected ec2 instances
func (ei *EC2Instances) filterOutProtectedInstances(output *ec2.DescribeInstancesOutput, configObj config.Config) ([]*string, error) {
	var filteredIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId

			attr, err := ei.Client.DescribeInstanceAttribute(ei.Context, &ec2.DescribeInstanceAttributeInput{
				Attribute:  types.InstanceAttributeNameDisableApiTermination,
				InstanceId: aws.String(instanceID),
			})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if shouldIncludeInstanceId(instance, *attr.DisableApiTermination.Value, configObj) {
				filteredIds = append(filteredIds, &instanceID)
			}
		}
	}

	return filteredIds, nil
}

// Returns a formatted string of EC2 instance ids
func (ei *EC2Instances) getAll(ctx context.Context, configObj config.Config) ([]*string, error) {
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

	output, err := ei.Client.DescribeInstances(ctx, params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	instanceIds, err := ei.filterOutProtectedInstances(output, configObj)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return instanceIds, nil
}

func shouldIncludeInstanceId(instance types.Instance, protected bool, configObj config.Config) bool {
	if protected {
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	instanceName := util.GetEC2ResourceNameTagValue(instance.Tags)
	return configObj.EC2.ShouldInclude(config.ResourceValue{
		Name: instanceName,
		Time: instance.LaunchTime,
		Tags: util.ConvertTypesTagsToMap(instance.Tags),
	})
}

func (ei *EC2Instances) releaseEIPs(instanceIds []*string) error {
	logging.Debugf("Releasing Elastic IP address(s) associated with instances")

	for _, instanceID := range instanceIds {
		// Get the Elastic IPs associated with the EC2 instances
		output, err := ei.Client.DescribeAddresses(ei.Context, &ec2.DescribeAddressesInput{
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

			_, err := ei.Client.ReleaseAddress(ei.Context, &ec2.ReleaseAddressInput{
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

// Deletes all non protected EC2 instances
func (ei *EC2Instances) nukeAll(instanceIds []*string) error {
	if len(instanceIds) == 0 {
		logging.Debugf("No EC2 instances to nuke in region %s", ei.Region)
		return nil
	}

	// release the attached elastic ip's
	// Note: This should be done before terminating the EC2 instances
	err := ei.releaseEIPs(instanceIds)
	if err != nil {
		logging.Debugf("[Failed EIP release] %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("Terminating all EC2 instances in region %s", ei.Region)

	params := &ec2.TerminateInstancesInput{
		InstanceIds: aws.ToStringSlice(instanceIds),
	}

	_, err = ei.Client.TerminateInstances(ei.Context, params)
	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	waiter := ec2.NewInstanceTerminatedWaiter(ei.Client)
	err = waiter.Wait(ei.Context, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: aws.ToStringSlice(instanceIds),
			},
		},
	}, 15*time.Minute)

	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[OK] %d instance(s) terminated in %s", len(instanceIds), ei.Region)
	return nil
}

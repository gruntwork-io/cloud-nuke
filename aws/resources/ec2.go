package resources

import (
	"context"
	"github.com/gruntwork-io/cloud-nuke/util"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// returns only instance Ids of unprotected ec2 instances
func (ei *EC2Instances) filterOutProtectedInstances(output *ec2.DescribeInstancesOutput, configObj config.Config) ([]*string, error) {
	var filteredIds []*string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := *instance.InstanceId

			attr, err := ei.Client.DescribeInstanceAttributeWithContext(ei.Context, &ec2.DescribeInstanceAttributeInput{
				Attribute:  awsgo.String("disableApiTermination"),
				InstanceId: awsgo.String(instanceID),
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
func (ei *EC2Instances) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: awsgo.String("instance-state-name"),
				Values: []*string{
					awsgo.String("running"), awsgo.String("pending"),
					awsgo.String("stopped"), awsgo.String("stopping"),
				},
			},
		},
	}

	output, err := ei.Client.DescribeInstancesWithContext(ei.Context, params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	instanceIds, err := ei.filterOutProtectedInstances(output, configObj)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return instanceIds, nil
}

func shouldIncludeInstanceId(instance *ec2.Instance, protected bool, configObj config.Config) bool {
	if protected {
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	instanceName := util.GetEC2ResourceNameTagValue(instance.Tags)
	return configObj.EC2.ShouldInclude(config.ResourceValue{
		Name: instanceName,
		Time: instance.LaunchTime,
		Tags: util.ConvertEC2TagsToMap(instance.Tags),
	})
}

func (ei *EC2Instances) releaseEIPs(instanceIds []*string) error {
	logging.Debugf("Releasing Elastic IP address(s) associated on instances")
	for _, instanceID := range instanceIds {

		// get the elastic ip's associated with the EC2's
		output, err := ei.Client.DescribeAddressesWithContext(ei.Context, &ec2.DescribeAddressesInput{
			Filters: []*ec2.Filter{
				{
					Name: awsgo.String("instance-id"),
					Values: []*string{
						instanceID,
					},
				},
			},
		})
		if err != nil {
			return err
		}

		for _, address := range output.Addresses {
			_, err := ei.Client.ReleaseAddressWithContext(ei.Context, &ec2.ReleaseAddressInput{
				AllocationId: address.AllocationId,
			})

			if err != nil {
				logging.Debugf("An error happened while releasing the elastic ip address %s, error %v", *address.AllocationId, err)
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
		InstanceIds: instanceIds,
	}

	_, err = ei.Client.TerminateInstancesWithContext(ei.Context, params)
	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	err = ei.Client.WaitUntilInstanceTerminatedWithContext(ei.Context, &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("instance-id"),
				Values: instanceIds,
			},
		},
	})
	for _, instanceID := range instanceIds {
		logging.Debugf("Terminated EC2 Instance: %s", *instanceID)
	}

	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[OK] %d instance(s) terminated in %s", len(instanceIds), ei.Region)
	return nil
}

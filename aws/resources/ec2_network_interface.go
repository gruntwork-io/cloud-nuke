package resources

import (
	"context"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeNetworkInterface(networkInterface *ec2.NetworkInterface, firstSeenTime *time.Time, configObj config.Config) bool {
	var interfaceName string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(networkInterface.TagSet)
	if name, ok := tagMap["Name"]; ok {
		interfaceName = name
	}
	return configObj.NetworkInterface.ShouldInclude(config.ResourceValue{
		Name: &interfaceName,
		Tags: tagMap,
		Time: firstSeenTime,
	})
}

func (ni *NetworkInterface) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string

	var firstSeenTime *time.Time
	var err error

	resp, err := ni.Client.DescribeNetworkInterfacesWithContext(ni.Context, &ec2.DescribeNetworkInterfacesInput{})
	if err != nil {
		logging.Debugf("[Internet Gateway] Failed to list internet gateways: %s", err)
		return nil, err
	}

	for _, networkInterface := range resp.NetworkInterfaces {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, ni.Client, networkInterface.NetworkInterfaceId, util.ConvertEC2TagsToMap(networkInterface.TagSet))
		if err != nil {
			logging.Error("unable to retrieve first seen tag")
			continue
		}

		if shouldIncludeNetworkInterface(networkInterface, firstSeenTime, configObj) {
			identifiers = append(identifiers, networkInterface.NetworkInterfaceId)
		}
	}

	// Check and verify the list of allowed nuke actions
	ni.VerifyNukablePermissions(identifiers, func(id *string) error {
		_, err := ni.Client.DeleteNetworkInterfaceWithContext(ni.Context, &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: id,
			DryRun:             awsgo.Bool(true),
		})
		return err
	})

	return identifiers, nil
}

func (ni *NetworkInterface) nuke(id *string) error {
	// detach network interfaces
	if err := ni.detachNetworkInterface(id); err != nil {
		return errors.WithStackTrace(err)
	}
	// nuking the network interface
	if err := nukeNetworkInterface(ni.Client, id); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (ni *NetworkInterface) detachNetworkInterface(id *string) error {
	logging.Debugf("Detaching network interface %s from instances ", awsgo.StringValue(id))

	output, err := ni.Client.DescribeNetworkInterfacesWithContext(ni.Context, &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{id},
	})
	if err != nil {
		logging.Debugf("[Failed] Error describing network interface %s: %s", awsgo.StringValue(id), err)
		return errors.WithStackTrace(err)
	}

	for _, networkInterface := range output.NetworkInterfaces {

		// check there has some attachments
		if networkInterface.Attachment == nil {
			continue
		}

		// nuking the attached instance
		// this will also remove the network interface
		if err := ni.nukeInstance(networkInterface.Attachment.InstanceId); err != nil {
			logging.Debugf("[Failed] Error nuking the attached instance %s on network interface %s %s", awsgo.StringValue(networkInterface.Attachment.InstanceId), awsgo.StringValue(id), err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] successfully detached network interface associated on instances")

	return nil
}

func (ni *NetworkInterface) releaseEIPs(instance *string) error {
	logging.Debugf("Releasing Elastic IP address(s) associated on instance %s", awsgo.StringValue(instance))
	// get the elastic ip's associated with the EC2's
	output, err := ni.Client.DescribeAddressesWithContext(ni.Context, &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				Name: awsgo.String("instance-id"),
				Values: []*string{
					instance,
				},
			},
		},
	})
	if err != nil {
		return err
	}

	for _, address := range output.Addresses {
		if _, err := ni.Client.ReleaseAddressWithContext(ni.Context, &ec2.ReleaseAddressInput{
			AllocationId: address.AllocationId,
		}); err != nil {
			logging.Debugf("An error happened while releasing the elastic ip address %s, error %v", awsgo.StringValue(address.AllocationId), err)
			continue
		}

		logging.Debugf("Released Elastic IP address %s from instance %s", awsgo.StringValue(address.AllocationId), awsgo.StringValue(instance))
	}

	logging.Debugf("[OK] successfully released Elastic IP address(s) associated on instances")

	return nil
}

func (ni *NetworkInterface) nukeInstance(id *string) error {

	// Needs to release the elastic ips attached on the instance before nuking
	if err := ni.releaseEIPs(id); err != nil {
		logging.Debugf("[Failed EIP release] %s", err)
		return errors.WithStackTrace(err)
	}

	// terminating the instance
	if _, err := ni.Client.TerminateInstancesWithContext(ni.Context, &ec2.TerminateInstancesInput{
		InstanceIds: []*string{id},
	}); err != nil {
		logging.Debugf("[Failed] Ec2 termination %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[Instance Termination] waiting to terminate instance %s", awsgo.StringValue(id))

	// wait until the instance terminated.
	if err := ni.Client.WaitUntilInstanceTerminatedWithContext(ni.Context, &ec2.DescribeInstancesInput{
		InstanceIds: []*string{id},
	}); err != nil {
		logging.Debugf("[Instance Termination Waiting] Failed to terminate instance %s : %s", *id, err)
		return errors.WithStackTrace(err)
	}
	logging.Debugf("[OK] successfully nuked instance %v", awsgo.StringValue(id))

	return nil
}

func (ni *NetworkInterface) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No network interface identifiers to nuke in region %s", ni.Region)
		return nil
	}

	logging.Debugf("Deleting all network interface in region %s", ni.Region)
	var deleted []*string

	for _, id := range identifiers {
		if nukable, reason := ni.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, reason)
			continue
		}

		err := ni.nuke(id)

		// Record status of this resource
		e := report.Entry{ // Use the 'r' alias to refer to the package
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Network Interface",
			Error:        err,
		}
		report.Record(e)

		if err == nil {
			deleted = append(deleted, id)
		}
	}

	logging.Debugf("[OK] %d network interface(s) deleted in %s", len(deleted), ni.Region)

	return nil
}

func nukeNetworkInterface(client ec2iface.EC2API, id *string) error {
	logging.Debugf("Deleting network interface %s", awsgo.StringValue(id))

	// If the network interface was attached to an instance, then when we remove the instance above, the network interface will also be removed.
	// However, when we attempt to nuke the interface here, we may encounter an error such as InvalidNetworkInterfaceID.NotFound.
	// In other situations, such as when the network interface hasn't been attached to any instances, we won't encounter this error.
	//
	// Note: We are handling the situation here by checking the error response from AWS.

	_, err := client.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: id,
	})

	// check the error exists and it is not the interfaceid not found
	if err != nil && util.TransformAWSError(err) != util.ErrInterfaceIDNotFound {
		logging.Debugf("[Failed] Error deleting network interface %s: %s", awsgo.StringValue(id), err)
		return errors.WithStackTrace(err)
	}
	logging.Debugf("[Ok] network interface deleted successfully %s", awsgo.StringValue(id))

	return nil
}

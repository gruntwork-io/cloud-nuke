package resources

import (
	"context"
	"fmt"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	r "github.com/gruntwork-io/cloud-nuke/report" // Alias the package as 'r'
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeGateway(ig *ec2.InternetGateway, firstSeenTime *time.Time, configObj config.Config) bool {
	var internetGateway string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(ig.Tags)
	if name, ok := tagMap["Name"]; ok {
		internetGateway = name
	}

	return configObj.InternetGateway.ShouldInclude(config.ResourceValue{
		Name: &internetGateway,
		Tags: tagMap,
		Time: firstSeenTime,
	})
}

func (igw *InternetGateway) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
	var firstSeenTime *time.Time
	var err error

	input := &ec2.DescribeInternetGatewaysInput{}
	resp, err := igw.Client.DescribeInternetGatewaysWithContext(igw.Context, input)
	if err != nil {
		logging.Debugf("[Internet Gateway] Failed to list internet gateways: %s", err)
		return nil, err
	}
	for _, ig := range resp.InternetGateways {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, igw.Client, ig.InternetGatewayId, util.ConvertEC2TagsToMap(ig.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if shouldIncludeGateway(ig, firstSeenTime, configObj) {
			identifiers = append(identifiers, ig.InternetGatewayId)

			// get vpc id for this igw and update the map
			if len(ig.Attachments) > 0 {
				igw.GatewayVPCMap[awsgo.StringValue(ig.InternetGatewayId)] = awsgo.StringValue(ig.Attachments[0].VpcId)
			}
		}
	}

	// Check and verify the list of allowed nuke actions
	igw.VerifyNukablePermissions(identifiers, func(id *string) error {
		params := &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: id,
			DryRun:            awsgo.Bool(true),
		}
		_, err := igw.Client.DeleteInternetGatewayWithContext(igw.Context, params)
		return err
	})

	return identifiers, nil
}

func (igw *InternetGateway) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No internet gateway identifiers to nuke in region %s", igw.Region)
		return nil
	}

	logging.Debugf("Deleting all internet gateways in region %s", igw.Region)
	var deletedGateways []*string

	for _, id := range identifiers {
		if nukable, reason := igw.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, reason)
			continue
		}

		err := igw.nuke(id)
		// Record status of this resource
		e := r.Entry{ // Use the 'r' alias to refer to the package
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Internet Gateway",
			Error:        err,
		}
		r.Record(e)

		if err == nil {
			deletedGateways = append(deletedGateways, id)
		}
	}

	logging.Debugf("[OK] %d internet gateway(s) deleted in %s", len(deletedGateways), igw.Region)

	return nil
}

func (igw *InternetGateway) nuke(id *string) error {
	// get the vpc id for current igw
	vpcID, ok := igw.GatewayVPCMap[awsgo.StringValue(id)]
	if !ok {
		logging.Debug(fmt.Sprintf("Failed to read the vpc Id for %s",
			awsgo.StringValue(id)))
		return fmt.Errorf("Failed to retrieve the VPC ID for %s, which is mandatory for the internet gateway nuke operation.",
			awsgo.StringValue(id))
	}

	err := nukeInternetGateway(igw.Client, id, vpcID)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func nukeInternetGateway(client ec2iface.EC2API, gatewayId *string, vpcID string) error {
	var err error
	logging.Debug(fmt.Sprintf("Detaching Internet Gateway %s",
		awsgo.StringValue(gatewayId)))
	_, err = client.DetachInternetGateway(
		&ec2.DetachInternetGatewayInput{
			InternetGatewayId: gatewayId,
			VpcId:             awsgo.String(vpcID),
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to detach internet gateway %s",
			awsgo.StringValue(gatewayId)))
		return errors.WithStackTrace(err)
	}
	logging.Debug(fmt.Sprintf("Successfully detached internet gateway %s",
		awsgo.StringValue(gatewayId)))

	// nuking the internet gateway
	logging.Debug(fmt.Sprintf("Deleting internet gateway %s",
		awsgo.StringValue(gatewayId)))
	_, err = client.DeleteInternetGateway(
		&ec2.DeleteInternetGatewayInput{
			InternetGatewayId: gatewayId,
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete internet gateway %s",
			awsgo.StringValue(gatewayId)))
		return errors.WithStackTrace(err)
	}
	logging.Debug(fmt.Sprintf("Successfully deleted internet gateway %s",
		awsgo.StringValue(gatewayId)))

	return nil

}

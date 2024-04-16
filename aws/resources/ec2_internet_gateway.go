package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	r "github.com/gruntwork-io/cloud-nuke/report" // Alias the package as 'r'
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (igw *InternetGateway) setFirstSeenTag(gateway ec2.InternetGateway, value time.Time) error {
	_, err := igw.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{gateway.InternetGatewayId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String(util.FirstSeenTagKey),
				Value: awsgo.String(util.FormatTimestamp(value)),
			},
		},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (igw *InternetGateway) getFirstSeenTag(gateway ec2.InternetGateway) (*time.Time, error) {
	tags := gateway.Tags
	for _, tag := range tags {
		if util.IsFirstSeenTag(tag.Key) {
			firstSeenTime, err := util.ParseTimestamp(tag.Value)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return nil, nil
}

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

func (igw *InternetGateway) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string

	input := &ec2.DescribeInternetGatewaysInput{}
	resp, err := igw.Client.DescribeInternetGateways(input)
	if err != nil {
		logging.Debugf("[Internet Gateway] Failed to list internet gateways: %s", err)
		return nil, err
	}

	for _, ig := range resp.InternetGateways {
		// check first seen tag
		firstSeenTime, err := igw.getFirstSeenTag(*ig)
		if err != nil {
			logging.Errorf(
				"Unable to retrieve tags for Internet gateway: %s, with error: %s", *ig.InternetGatewayId, err)
			continue
		}

		// if the first seen tag is not there, then create one
		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := igw.setFirstSeenTag(*ig, time.Now().UTC()); err != nil {
				logging.Errorf(
					"Unable to apply first seen tag Internet gateway: %s, with error: %s", *ig.InternetGatewayId, err)
				continue
			}
		}

		if shouldIncludeGateway(ig, firstSeenTime, configObj) {
			identifiers = append(identifiers, ig.InternetGatewayId)
		}
	}

	// Check and verify the list of allowed nuke actions
	igw.VerifyNukablePermissions(identifiers, func(id *string) error {
		params := &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: id,
			DryRun:            aws.Bool(true),
		}
		_, err := igw.Client.DeleteInternetGateway(params)
		return err
	})

	return identifiers, nil
}

func nukeInternetGateway(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking Internet Gateway for vpc: %s", vpcID))
	input := &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("attachment.vpc-id"),
				Values: []*string{awsgo.String(vpcID)},
			},
		},
	}
	igw, err := client.DescribeInternetGateways(input)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe internet gateways for vpc: %s", vpcID))
		return errors.WithStackTrace(err)
	}

	if len(igw.InternetGateways) < 1 {
		logging.Debug(fmt.Sprintf("No Internet Gateway to delete."))
		return nil
	}

	logging.Debug(fmt.Sprintf("Detaching Internet Gateway %s",
		awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId)))
	_, err = client.DetachInternetGateway(
		&ec2.DetachInternetGatewayInput{
			InternetGatewayId: igw.InternetGateways[0].InternetGatewayId,
			VpcId:             awsgo.String(vpcID),
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to detach internet gateway %s",
			awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId)))
		return errors.WithStackTrace(err)
	}
	logging.Debug(fmt.Sprintf("Successfully detached internet gateway %s",
		awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId)))

	logging.Debug(fmt.Sprintf("Deleting internet gateway %s",
		awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId)))
	_, err = client.DeleteInternetGateway(
		&ec2.DeleteInternetGatewayInput{
			InternetGatewayId: igw.InternetGateways[0].InternetGatewayId,
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete internet gateway %s",
			awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId)))
		return errors.WithStackTrace(err)
	}
	logging.Debug(fmt.Sprintf("Successfully deleted internet gateway %s",
		awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId)))

	return nil
}

func (igw *InternetGateway) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No internet gateway identifiers to nuke in region %s", igw.Region)
		return nil
	}

	logging.Debugf("Deleting all internet gateways in region %s", igw.Region)
	var deletedGateways []*string

	for _, id := range identifiers {
		if nukable, err := igw.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		err := nuke(igw.Client, *id)
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

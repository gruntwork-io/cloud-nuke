package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
				Key:   aws.String(util.FirstSeenTagKey),
				Value: aws.String(util.FormatTimestamp(value)),
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

func (igw *InternetGateway) nuke(id *string) error {

	// detaching the gateway from attached vpcs
	if err := igw.detachInternetGateway(id); err != nil {
		return errors.WithStackTrace(err)
	}

	// nuking the gateway
	if err := igw.nukeInternetGateway(id); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (igw *InternetGateway) nukeInternetGateway(id *string) error {
	logging.Debugf("Deleting Internet gateway %s", *id)

	_, err := igw.Client.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
		InternetGatewayId: id,
	})
	if err != nil {
		logging.Debugf("[Failed] Error deleting internet gateway %s: %s", *id, err)
		return errors.WithStackTrace(err)
	}
	logging.Debugf("[Ok] internet gateway deleted successfully %s", *id)

	return nil
}

func (igw *InternetGateway) detachInternetGateway(id *string) error {
	logging.Debugf("Detaching Internet gateway %s", *id)

	output, err := igw.Client.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []*string{id},
	})
	if err != nil {
		logging.Debugf("[Failed] Error describing internet gateway %s: %s", *id, err)
		return errors.WithStackTrace(err)
	}

	for _, gateway := range output.InternetGateways {
		for _, attachments := range gateway.Attachments {
			_, err := igw.Client.DetachInternetGateway(&ec2.DetachInternetGatewayInput{
				InternetGatewayId: id,
				VpcId:             attachments.VpcId,
			})
			if err != nil {
				logging.Debugf("[Failed] Error detaching internet gateway %s: %s", *id, err)
				return errors.WithStackTrace(err)
			}
		}
	}
	logging.Debugf("[Ok] internet gateway detached successfully %s", *id)
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

		err := igw.nuke(id)
		// Record status of this resource
		e := r.Entry{ // Use the 'r' alias to refer to the package
			Identifier:   aws.StringValue(id),
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

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
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (egigw *EgressOnlyInternetGateway) setFirstSeenTag(eoig ec2.EgressOnlyInternetGateway, value time.Time) error {
	_, err := egigw.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{eoig.EgressOnlyInternetGatewayId},
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

func (egigw *EgressOnlyInternetGateway) getFirstSeenTag(eoig ec2.EgressOnlyInternetGateway) (*time.Time, error) {
	tags := eoig.Tags
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

func shouldIncludeEgressOnlyInternetGateway(gateway *ec2.EgressOnlyInternetGateway, firstSeenTime *time.Time, configObj config.Config) bool {
	var gatewayName string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(gateway.Tags)
	if name, ok := tagMap["Name"]; ok {
		gatewayName = name
	}
	return configObj.EgressOnlyInternetGateway.ShouldInclude(config.ResourceValue{
		Name: &gatewayName,
		Tags: tagMap,
		Time: firstSeenTime,
	})
}

func (egigw *EgressOnlyInternetGateway) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var result []*string

	output, err := egigw.Client.DescribeEgressOnlyInternetGateways(&ec2.DescribeEgressOnlyInternetGatewaysInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, igw := range output.EgressOnlyInternetGateways {
		// check first seen tag
		firstSeenTime, err := egigw.getFirstSeenTag(*igw)
		if err != nil {
			logging.Errorf(
				"Unable to retrieve tags for Egress IGW: %s, with error: %s", *igw.EgressOnlyInternetGatewayId, err)
			continue
		}

		// if the first seen tag is not there, then create one
		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := egigw.setFirstSeenTag(*igw, time.Now().UTC()); err != nil {
				logging.Errorf(
					"Unable to apply first seen tag Egress IGW: %s, with error: %s", *igw.EgressOnlyInternetGatewayId, err)
				continue
			}
		}

		if shouldIncludeEgressOnlyInternetGateway(igw, firstSeenTime, configObj) {
			result = append(result, igw.EgressOnlyInternetGatewayId)
		}
	}

	// checking the nukable permissions
	egigw.VerifyNukablePermissions(result, func(id *string) error {
		_, err := egigw.Client.DeleteEgressOnlyInternetGateway(&ec2.DeleteEgressOnlyInternetGatewayInput{
			EgressOnlyInternetGatewayId: id,
			DryRun:                      awsgo.Bool(true),
		})
		return err
	})

	return result, nil
}

func (egigw *EgressOnlyInternetGateway) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Egress only internet gateway ID's to nuke in region %s", egigw.Region)
		return nil
	}

	logging.Debugf("Deleting all Egress only internet gateway in region %s", egigw.Region)
	var deletedList []*string

	for _, id := range ids {
		// NOTE : We can skip the error checking and return it here, since it is already being checked while displaying the identifiers with the Nukable  field.
		// Here, `err` refers to the error indicating whether the identifier is eligible for nuke or not (an error which we got from aws when tried to delete the resource with dryRun),
		// and it is not a programming error. (edited)
		if nukable, err := egigw.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		err := nukeEgressOnlyGateway(egigw.Client, id)

		// Record status of this resource
		e := report.Entry{
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Egress Only Internet Gateway",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedList = append(deletedList, id)
			logging.Debugf("Deleted egress only internet gateway: %s", *id)
		}
	}

	logging.Debugf("[OK] %d Egress only internet gateway(s) deleted in %s", len(deletedList), egigw.Region)

	return nil

}

func nukeEgressOnlyGateway(client ec2iface.EC2API, gateway *string) error {
	logging.Debugf("[Nuke] Egress only gateway %s", awsgo.StringValue(gateway))

	_, err := client.DeleteEgressOnlyInternetGateway(&ec2.DeleteEgressOnlyInternetGatewayInput{
		EgressOnlyInternetGatewayId: gateway,
	})
	if err != nil {
		logging.Debug(fmt.Sprintf("[Failed] to delete Egress Only Internet Gateway %s", *gateway))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("[Success] deleted Egress Only Internet Gateway %s", *gateway))
	return nil
}

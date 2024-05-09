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

func (egigw *EgressOnlyInternetGateway) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var result []*string
	var firstSeenTime *time.Time
	var err error

	output, err := egigw.Client.DescribeEgressOnlyInternetGatewaysWithContext(egigw.Context, &ec2.DescribeEgressOnlyInternetGatewaysInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, igw := range output.EgressOnlyInternetGateways {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, egigw.Client, igw.EgressOnlyInternetGatewayId, util.ConvertEC2TagsToMap(igw.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
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
		return errors.WithStackTrace(err)
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

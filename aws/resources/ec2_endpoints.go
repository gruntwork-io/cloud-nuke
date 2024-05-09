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
	"github.com/gruntwork-io/go-commons/retry"
)

func ShouldIncludeVpcEndpoint(endpoint *ec2.VpcEndpoint, firstSeenTime *time.Time, configObj config.Config) bool {
	var endpointName string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(endpoint.Tags)
	if name, ok := tagMap["Name"]; ok {
		endpointName = name
	}

	return configObj.EC2Endpoint.ShouldInclude(config.ResourceValue{
		Name: &endpointName,
		Time: firstSeenTime,
		Tags: tagMap,
	})
}

func (e *EC2Endpoints) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var result []*string

	var firstSeenTime *time.Time
	var err error
	endpoints, err := e.Client.DescribeVpcEndpointsWithContext(e.Context, &ec2.DescribeVpcEndpointsInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, endpoint := range endpoints.VpcEndpoints {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, e.Client, endpoint.VpcEndpointId, util.ConvertEC2TagsToMap(endpoint.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if ShouldIncludeVpcEndpoint(endpoint, firstSeenTime, configObj) {
			result = append(result, endpoint.VpcEndpointId)
		}

	}

	e.VerifyNukablePermissions(result, func(id *string) error {
		_, err := e.Client.DeleteVpcEndpoints(&ec2.DeleteVpcEndpointsInput{
			VpcEndpointIds: []*string{id},
			DryRun:         awsgo.Bool(true),
		})
		return err
	})

	return result, nil
}

func (e *EC2Endpoints) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Vpc Endpoints to nuke in region %s", e.Region)
		return nil
	}

	logging.Debugf("Deleting all Vpc Endpoints in region %s", e.Region)
	var deletedAddresses []*string

	for _, id := range identifiers {
		if nukable, err := e.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		err := nukeVpcEndpoint(e.Client, []*string{id})

		// Record status of this resource
		e := report.Entry{
			Identifier:   awsgo.StringValue(id),
			ResourceType: "Vpc Endpoint",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedAddresses = append(deletedAddresses, id)
		}
	}

	logging.Debugf("[OK] %d Vpc Endpoint(s) deleted in %s", len(deletedAddresses), e.Region)

	return nil
}

func nukeVpcEndpoint(client ec2iface.EC2API, endpointIds []*string) error {
	logging.Debugf("Deleting VPC endpoints %s", awsgo.StringValueSlice(endpointIds))

	_, err := client.DeleteVpcEndpoints(&ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: endpointIds,
	})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete VPC endpoints: %s", err.Error()))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Successfully deleted VPC endpoints %s",
		awsgo.StringValueSlice(endpointIds)))

	return nil
}

func waitForVPCEndpointToBeDeleted(client ec2iface.EC2API, vpcID string) error {
	return retry.DoWithRetry(
		logging.Logger.WithTime(time.Now()),
		"Waiting for all VPC endpoints to be deleted",
		10,
		2*time.Second,
		func() error {
			endpoints, err := client.DescribeVpcEndpoints(
				&ec2.DescribeVpcEndpointsInput{
					Filters: []*ec2.Filter{
						{
							Name:   awsgo.String("vpc-id"),
							Values: []*string{awsgo.String(vpcID)},
						},
						{
							Name:   awsgo.String("vpc-endpoint-state"),
							Values: []*string{awsgo.String("deleting")},
						},
					},
				},
			)
			if err != nil {
				return err
			}

			if len(endpoints.VpcEndpoints) == 0 {
				return nil
			}
			return fmt.Errorf("Not all VPC endpoints deleted.")
		},
	)
	return nil
}

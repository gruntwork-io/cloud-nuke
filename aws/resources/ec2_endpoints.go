package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func ShouldIncludeVpcEndpoint(endpoint *types.VpcEndpoint, firstSeenTime *time.Time, configObj config.Config) bool {
	var endpointName string
	// get the tags as map
	tagMap := util.ConvertTypesTagsToMap(endpoint.Tags)
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
	endpoints, err := e.Client.DescribeVpcEndpoints(e.Context, &ec2.DescribeVpcEndpointsInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, endpoint := range endpoints.VpcEndpoints {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, e.Client, endpoint.VpcEndpointId, util.ConvertTypesTagsToMap(endpoint.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if ShouldIncludeVpcEndpoint(&endpoint, firstSeenTime, configObj) {
			result = append(result, endpoint.VpcEndpointId)
		}

	}

	e.VerifyNukablePermissions(result, func(id *string) error {
		_, err := e.Client.DeleteVpcEndpoints(e.Context, &ec2.DeleteVpcEndpointsInput{
			VpcEndpointIds: []string{aws.ToString(id)},
			DryRun:         aws.Bool(true),
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
		if nukable, reason := e.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, reason)
			continue
		}

		err := e.nukeVpcEndpoint([]*string{id})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
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

func (e *EC2Endpoints) nukeVpcEndpoint(endpointIds []*string) error {
	logging.Debugf("Deleting VPC endpoints %s", aws.ToStringSlice(endpointIds))

	_, err := e.Client.DeleteVpcEndpoints(e.Context, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: aws.ToStringSlice(endpointIds),
	})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete VPC endpoints: %s", err.Error()))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Successfully deleted VPC endpoints %s", aws.ToStringSlice(endpointIds)))

	return nil
}

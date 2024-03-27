package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (e *EC2Endpoints) setFirstSeenTag(endpoint ec2.VpcEndpoint, value time.Time) error {
	_, err := e.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{endpoint.VpcEndpointId},
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

func (e *EC2Endpoints) getFirstSeenTag(endpoint ec2.VpcEndpoint) (*time.Time, error) {
	for _, tag := range endpoint.Tags {
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

func (e *EC2Endpoints) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var result []*string
	endpoints, err := e.Client.DescribeVpcEndpoints(&ec2.DescribeVpcEndpointsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, endpoint := range endpoints.VpcEndpoints {
		// check first seen tag
		firstSeenTime, err := e.getFirstSeenTag(*endpoint)
		if err != nil {
			logging.Errorf(
				"Unable to retrieve tags for Vpc Endpoint: %s, with error: %s", *endpoint.VpcEndpointId, err)
			continue
		}

		// if the first seen tag is not there, then create one
		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := e.setFirstSeenTag(*endpoint, time.Now().UTC()); err != nil {
				logging.Errorf(
					"Unable to apply first seen tag Vpc Endpoint: %s, with error: %s", *endpoint.VpcEndpointId, err)
				continue
			}
		}

		if ShouldIncludeVpcEndpoint(endpoint, firstSeenTime, configObj) {
			result = append(result, endpoint.VpcEndpointId)
		}
	}

	e.VerifyNukablePermissions(result, func(id *string) error {
		_, err := e.Client.DeleteVpcEndpoints(&ec2.DeleteVpcEndpointsInput{
			VpcEndpointIds: []*string{id},
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
		if nukable, err := e.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		_, err := e.Client.DeleteVpcEndpoints(&ec2.DeleteVpcEndpointsInput{
			VpcEndpointIds: []*string{id},
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "Vpc Endpoint",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedAddresses = append(deletedAddresses, id)
			logging.Debugf("Deleted Vpc Endpoint: %s", *id)
		}
	}

	logging.Debugf("[OK] %d Vpc Endpoint(s) deleted in %s", len(deletedAddresses), e.Region)

	return nil
}

package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (scope *EC2IpamScopes) setFirstSeenTag(ipam ec2.IpamScope, value time.Time) error {
	_, err := scope.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{ipam.IpamScopeId},
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

func (scope *EC2IpamScopes) getFirstSeenTag(ipam ec2.IpamScope) (*time.Time, error) {
	tags := ipam.Tags
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

func shouldIncludeIpamScopeID(ipam *ec2.IpamScope, firstSeenTime *time.Time, configObj config.Config) bool {
	var ipamScopeName string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(ipam.Tags)
	if name, ok := tagMap["Name"]; ok {
		ipamScopeName = name
	}

	return configObj.EC2IPAMScope.ShouldInclude(config.ResourceValue{
		Name: &ipamScopeName,
		Time: firstSeenTime,
		Tags: tagMap,
	})
}

// Returns a formatted string of IPAM URLs
func (ec2Scope *EC2IpamScopes) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result := []*string{}
	paginator := func(output *ec2.DescribeIpamScopesOutput, lastPage bool) bool {
		for _, scope := range output.IpamScopes {
			// check first seen tag
			firstSeenTime, err := ec2Scope.getFirstSeenTag(*scope)
			if err != nil {
				logging.Errorf(
					"Unable to retrieve tags for IPAM: %s, with error: %s", *scope.IpamScopeId, err)
				continue
			}

			// if the first seen tag is not there, then create one
			if firstSeenTime == nil {
				now := time.Now().UTC()
				firstSeenTime = &now
				if err := ec2Scope.setFirstSeenTag(*scope, time.Now().UTC()); err != nil {
					logging.Errorf(
						"Unable to apply first seen tag IPAM: %s, with error: %s", *scope.IpamScopeId, err)
					continue
				}
			}
			// Check for include this ipam
			if shouldIncludeIpamScopeID(scope, firstSeenTime, configObj) {
				result = append(result, scope.IpamScopeId)
			}
		}
		return !lastPage
	}

	params := &ec2.DescribeIpamScopesInput{
		MaxResults: awsgo.Int64(10),
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("is-default"),
				Values: awsgo.StringSlice([]string{"false"}),
			},
		},
	}

	err := ec2Scope.Client.DescribeIpamScopesPages(params, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return result, nil
}

// Deletes all IPAMs
func (scope *EC2IpamScopes) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No IPAM Scopes ID's to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting all IPAM Scopes in region %s", scope.Region)
	var deletedList []*string

	for _, id := range ids {
		params := &ec2.DeleteIpamScopeInput{
			IpamScopeId: id,
		}

		_, err := scope.Client.DeleteIpamScope(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "IPAM Scopes",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedList = append(deletedList, id)
			logging.Debugf("Deleted IPAM Scope: %s", *id)
		}
	}

	logging.Debugf("[OK] %d IPAM Scope(s) deleted in %s", len(deletedList), scope.Region)

	return nil
}

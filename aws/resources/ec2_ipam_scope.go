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
	var firstSeenTime *time.Time
	var err error

	paginator := func(output *ec2.DescribeIpamScopesOutput, lastPage bool) bool {
		for _, scope := range output.IpamScopes {

			firstSeenTime, err = util.GetOrCreateFirstSeen(c, ec2Scope.Client, scope.IpamScopeId, util.ConvertEC2TagsToMap(scope.Tags))
			if err != nil {
				logging.Error("unable to retrieve firstseen tag")
				continue
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

	err = ec2Scope.Client.DescribeIpamScopesPages(params, paginator)
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

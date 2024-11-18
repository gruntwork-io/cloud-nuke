package resources

import (
	"context"
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

func shouldIncludeIpamScopeID(ipam *types.IpamScope, firstSeenTime *time.Time, configObj config.Config) bool {
	var ipamScopeName string
	// get the tags as map
	tagMap := util.ConvertTypesTagsToMap(ipam.Tags)
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
	var result []*string
	var firstSeenTime *time.Time
	var err error

	params := &ec2.DescribeIpamScopesInput{
		MaxResults: aws.Int32(10),
		Filters: []types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"false"},
			},
		},
	}

	scopesPaginator := ec2.NewDescribeIpamScopesPaginator(ec2Scope.Client, params)
	for scopesPaginator.HasMorePages() {
		page, errPaginator := scopesPaginator.NextPage(c)
		if errPaginator != nil {
			return nil, errors.WithStackTrace(errPaginator)
		}

		for _, scope := range page.IpamScopes {
			firstSeenTime, err = util.GetOrCreateFirstSeen(c, ec2Scope.Client, scope.IpamScopeId, util.ConvertTypesTagsToMap(scope.Tags))
			if err != nil {
				logging.Error("unable to retrieve firstseen tag")
				continue
			}

			// Check for include this ipam
			if shouldIncludeIpamScopeID(&scope, firstSeenTime, configObj) {
				result = append(result, scope.IpamScopeId)
			}
		}
	}

	// checking the nukable permissions
	ec2Scope.VerifyNukablePermissions(result, func(id *string) error {
		_, err := ec2Scope.Client.DeleteIpamScope(ec2Scope.Context, &ec2.DeleteIpamScopeInput{
			IpamScopeId: id,
			DryRun:      aws.Bool(true),
		})
		return err
	})

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
		if nukable, reason := scope.IsNukable(aws.ToString(id)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(id), reason)
			continue
		}

		_, err := scope.Client.DeleteIpamScope(scope.Context, &ec2.DeleteIpamScopeInput{
			IpamScopeId: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
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

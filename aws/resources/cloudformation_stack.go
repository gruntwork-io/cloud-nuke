package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (cfs *CloudFormationStacks) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := cfs.Client.ListStacks(c, &cloudformation.ListStacksInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var stackNames []*string
	for _, stack := range result.StackSummaries {
		// Get detailed stack information including tags
		stackDetails, err := cfs.Client.DescribeStacks(c, &cloudformation.DescribeStacksInput{
			StackName: stack.StackName,
		})
		if err != nil {
			logging.Debugf("Failed to describe stack %s: %v", *stack.StackName, err)
			continue
		}

		if len(stackDetails.Stacks) == 0 {
			continue
		}

		stackDetail := stackDetails.Stacks[0]
		tags := util.ConvertCloudFormationTagsToMap(stackDetail.Tags)

		if configObj.CloudFormationStack.ShouldInclude(config.ResourceValue{
			Name: stack.StackName,
			Time: stack.CreationTime,
			Tags: tags,
		}) {
			stackNames = append(stackNames, stack.StackName)
		}
	}

	return stackNames, nil
}

func (cfs *CloudFormationStacks) nukeAll(stackNames []*string) error {
	if len(stackNames) == 0 {
		logging.Debugf("No CloudFormation Stacks to nuke in region %s", cfs.Region)
		return nil
	}

	logging.Debugf("Deleting all CloudFormation Stacks in region %s", cfs.Region)
	var deletedStackNames []*string

	for _, stackName := range stackNames {
		_, err := cfs.Client.DeleteStack(cfs.Context, &cloudformation.DeleteStackInput{
			StackName: stackName,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(stackName),
			ResourceType: "CloudFormation Stack",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedStackNames = append(deletedStackNames, stackName)
			logging.Debugf("Deleted CloudFormation Stack: %s", aws.ToString(stackName))
		}
	}

	logging.Debugf("[OK] %d CloudFormation Stack(s) deleted in %s", len(deletedStackNames), cfs.Region)
	return nil
}

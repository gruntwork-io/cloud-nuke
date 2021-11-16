package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllCloudFormationStacks(session *session.Session) ([]*string, error) {
	svc := cloudformation.New(session)

	stacks, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, stack := range stacks.Stacks {
		ids = append(ids, stack.StackId)
	}

	return ids, nil
}

func getAllCloudFormationStacksSets(session *session.Session) ([]*string, error) {
	svc := cloudformation.New(session)

	stacks, err := svc.ListStackSets(&cloudformation.ListStackSetsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, set := range stacks.Summaries {
		ids = append(ids, set.StackSetId)
	}

	return ids, nil
}

func nukeAllCloudformationStacks(session *session.Session, ids []*string) error {
	if len(ids) == 0 {
		logging.Logger.Info("No Cloudformation Stacks to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all Cloudformation Stacks")

	deletedStacks := 0
	svc := cloudformation.New(session)
	multiErr := new(multierror.Error)

	for _, id := ids {
		if err := deleteCloudformation
	}

	return nil
}

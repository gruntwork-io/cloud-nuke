package aws

import (
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/retry"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllCloudFormationStacks(session *session.Session) ([]*string, error) {
	svc := cloudformation.New(session)

	stacks, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, stack := range stacks.Stacks {
		names = append(names, stack.StackName)
	}

	return names, nil
}

func nukeAllCloudformationStacks(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	if len(identifiers) == 0 {
		logging.Logger.Info("No Cloudformation Stacks to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all Cloudformation Stacks")

	svc := cloudformation.New(session)

	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, ngwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteCloudformationStackAsync(wg, errChans[i], svc, ngwID)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	err := retry.DoWithRetry(
		logging.Logger,
		"Waiting for all Cloudformation Stacks to be deleted.",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			areDeleted, err := areAllCloudformationStacksDeleted(svc, identifiers)
			if err != nil {
				return errors.WithStackTrace(retry.FatalError{Underlying: err})
			}
			if areDeleted {
				return nil
			}
			return fmt.Errorf("Not all Cloudformation Stacks deleted.")
		},
	)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	for _, stackName := range identifiers {
		logging.Logger.Infof("[OK] Cloudformation Stack %s was deleted in %s", aws.StringValue(stackName), region)
	}
	return nil
}

func deleteCloudformationStackAsync(wg *sync.WaitGroup, errChan chan error, svc *cloudformation.CloudFormation, stackName *string) {
	defer wg.Done()

	_, err := svc.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: stackName,
	})

	errChan <- err
}

func areAllCloudformationStacksDeleted(svc *cloudformation.CloudFormation, identifiers []*string) (bool, error) {
	resp, err := svc.ListStacks(&cloudformation.ListStacksInput{})
	if err != nil {
		return false, err
	}

	if len(resp.StackSummaries) == 0 {
		return true, nil
	}

	for _, stack := range resp.StackSummaries {
		if stack == nil {
			continue
		}

		if aws.StringValue(stack.StackStatus) != cloudformation.StackStatusDeleteComplete {
			return false, nil
		}
	}

	return false, nil
}

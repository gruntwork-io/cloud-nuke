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

func getAllCloudFormationStacksSets(session *session.Session) ([]*string, error) {
	svc := cloudformation.New(session)

	stacks, err := svc.ListStackSets(&cloudformation.ListStackSetsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, set := range stacks.Summaries {
		names = append(names, set.StackSetName)
	}

	return names, nil
}

func nukeAllCloudformationStackSets(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	if len(identifiers) == 0 {
		logging.Logger.Info("No Cloudformation Stack Sets to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all Cloudformation Stack Sets")

	svc := cloudformation.New(session)

	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, ngwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteCloudformationStackSetAsync(wg, errChans[i], svc, ngwID)
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
		"Waiting for all Cloudformation Stack Sets to be deleted.",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			areDeleted, err := areAllCloudformationStackSetsDeleted(svc, identifiers)
			if err != nil {
				return errors.WithStackTrace(retry.FatalError{Underlying: err})
			}
			if areDeleted {
				return nil
			}
			return fmt.Errorf("Not all Cloudformation Stack Sets deleted.")
		},
	)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	for _, setName := range identifiers {
		logging.Logger.Infof("[OK] Cloudformation Stack Set %s was deleted in %s", aws.StringValue(setName), region)
	}
	return nil
}

func deleteCloudformationStackSetAsync(wg *sync.WaitGroup, errChan chan error, svc *cloudformation.CloudFormation, stackSetName *string) {
	defer wg.Done()

	_, err := svc.DeleteStackSet(&cloudformation.DeleteStackSetInput{
		StackSetName: stackSetName,
	})

	errChan <- err
}

func areAllCloudformationStackSetsDeleted(svc *cloudformation.CloudFormation, identifiers []*string) (bool, error) {
	resp, err := svc.ListStackSets(&cloudformation.ListStackSetsInput{})
	if err != nil {
		return false, err
	}

	if len(resp.Summaries) == 0 {
		return true, nil
	}

	for _, set := range resp.Summaries {
		if set == nil {
			continue
		}

		if aws.StringValue(set.Status) != cloudformation.StackSetStatusDeleted {
			return false, nil
		}
	}

	return false, nil
}

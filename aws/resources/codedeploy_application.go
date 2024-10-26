package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (cda *CodeDeployApplications) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var codeDeployApplicationsFilteredByName []string

	paginator := codedeploy.NewListApplicationsPaginator(cda.Client, &codedeploy.ListApplicationsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, application := range page.Applications {
			// Check if the CodeDeploy Application should be excluded by name as that information is available to us here.
			// CreationDate is not available in the ListApplications API call, so we can't filter by that here, but we do filter by it later.
			// By filtering the name here, we can reduce the number of BatchGetApplication API calls we have to make.
			if configObj.CodeDeployApplications.ShouldInclude(config.ResourceValue{Name: aws.String(application)}) {
				codeDeployApplicationsFilteredByName = append(codeDeployApplicationsFilteredByName, application)
			}
		}
	}

	// Check if the CodeDeploy Application should be excluded by CreationDate and return.
	// We have to do this after the ListApplicationsPages API call because CreationDate is not available in that call.
	return cda.batchDescribeAndFilter(codeDeployApplicationsFilteredByName, configObj)
}

// batchDescribeAndFilterCodeDeployApplications - Describe the CodeDeploy Applications and filter out the ones that should be excluded by CreationDate.
func (cda *CodeDeployApplications) batchDescribeAndFilter(identifiers []string, configObj config.Config) ([]*string, error) {
	// BatchGetApplications can only take 100 identifiers at a time, so we have to break up the identifiers into chunks of 100.
	batchSize := 100
	var applicationNames []*string

	for {
		// if there are no identifiers left, then break out of the loop
		if len(identifiers) == 0 {
			break
		}

		// if the batch size is larger than the number of identifiers left, then set the batch size to the number of identifiers left
		if len(identifiers) < batchSize {
			batchSize = len(identifiers)
		}

		// get the next batch of identifiers
		batch := identifiers[:batchSize]
		// then using that batch of identifiers, get the applicationsinfo
		resp, err := cda.Client.BatchGetApplications(
			cda.Context,
			&codedeploy.BatchGetApplicationsInput{ApplicationNames: batch},
		)
		if err != nil {
			return nil, err
		}

		// for each applicationsinfo, check if it should be excluded by creation date
		for j := range resp.ApplicationsInfo {
			if configObj.CodeDeployApplications.ShouldInclude(config.ResourceValue{
				Time: resp.ApplicationsInfo[j].CreateTime,
			}) {
				applicationNames = append(applicationNames, resp.ApplicationsInfo[j].ApplicationName)
			}
		}

		// reduce the identifiers by the batch size we just processed, note that the slice header is mutated here
		identifiers = identifiers[batchSize:]
	}

	return applicationNames, nil
}

func (cda *CodeDeployApplications) nukeAll(identifiers []string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No CodeDeploy Applications to nuke in region %s", cda.Region)
		return nil
	}

	logging.Infof("Deleting CodeDeploy Applications in region %s", cda.Region)

	var wg sync.WaitGroup
	errChan := make(chan error, len(identifiers))

	for _, identifier := range identifiers {
		wg.Add(1)
		go cda.deleteAsync(&wg, errChan, identifier)
	}

	wg.Wait()
	close(errChan)

	var allErrors *multierror.Error
	for err := range errChan {
		allErrors = multierror.Append(allErrors, err)

		logging.Errorf("[Failed] Error deleting CodeDeploy Application: %s", err)
	}

	finalErr := allErrors.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	return nil
}

func (cda *CodeDeployApplications) deleteAsync(wg *sync.WaitGroup, errChan chan<- error, identifier string) {
	defer wg.Done()

	_, err := cda.Client.DeleteApplication(cda.Context, &codedeploy.DeleteApplicationInput{ApplicationName: &identifier})
	if err != nil {
		errChan <- err
	}

	// record the status of the nuke attempt
	e := report.Entry{
		Identifier:   identifier,
		ResourceType: "CodeDeploy Application",
		Error:        err,
	}
	report.Record(e)

	if err == nil {
		logging.Debugf("[OK] Deleted CodeDeploy Application: %s", identifier)
	} else {
		logging.Debugf("[Failed] Error deleting CodeDeploy Application %s: %s", identifier, err)
	}
}

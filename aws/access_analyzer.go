package aws

import (
	"github.com/aws/aws-sdk-go/service/accessanalyzer"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
	"sync"
)

func (analyzer AccessAnalyzer) getAll(configObj config.Config) ([]*string, error) {
	allAnalyzers := []*string{}
	err := analyzer.Client.ListAnalyzersPages(
		&accessanalyzer.ListAnalyzersInput{},
		func(page *accessanalyzer.ListAnalyzersOutput, lastPage bool) bool {
			for _, analyzer := range page.Analyzers {
				if configObj.AccessAnalyzer.ShouldInclude(config.ResourceValue{
					Time: analyzer.CreatedAt,
					Name: analyzer.Name,
				}) {
					allAnalyzers = append(allAnalyzers, analyzer.Name)
				}
			}
			return !lastPage
		},
	)
	return allAnalyzers, errors.WithStackTrace(err)
}

func (analyzer AccessAnalyzer) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Logger.Debugf("No IAM Access Analyzers to nuke in region %s", analyzer.Region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on AccessAnalyzer.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(names) > 100 {
		logging.Logger.Errorf("Nuking too many Access Analyzers at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyAccessAnalyzersErr{}
	}

	// There is no bulk delete access analyzer API, so we delete the batch of Access Analyzers concurrently using go routines.
	logging.Logger.Debugf("Deleting all Access Analyzers in region %s", analyzer.Region)

	wg := new(sync.WaitGroup)
	wg.Add(len(names))
	errChans := make([]chan error, len(names))
	for i, analyzerName := range names {
		errChans[i] = make(chan error, 1)
		go analyzer.deleteAccessAnalyzerAsync(wg, errChans[i], analyzerName)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Access Analyzer",
			}, map[string]interface{}{
				"region": analyzer.Region,
			})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	return errors.WithStackTrace(finalErr)
}

// deleteAccessAnalyzerAsync deletes the provided IAM Access Analyzer asynchronously in a goroutine, using wait groups
// for concurrency control and a return channel for errors.
func (analyzer AccessAnalyzer) deleteAccessAnalyzerAsync(wg *sync.WaitGroup, errChan chan error, analyzerName *string) {
	defer wg.Done()

	input := &accessanalyzer.DeleteAnalyzerInput{AnalyzerName: analyzerName}
	_, err := analyzer.Client.DeleteAnalyzer(input)
	errChan <- err
}

// Custom errors

type TooManyAccessAnalyzersErr struct{}

func (err TooManyAccessAnalyzersErr) Error() string {
	return "Too many Access Analyzers requested at once."
}

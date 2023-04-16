package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func getAllS3AccessPoints(session *session.Session, configObj config.Config) ([]*string, error) {
	svc := s3control.New(session)

	allAccessPoints := []*string{}

	accountId, err := util.GetCurrentAccountId(session)
	if err != nil {
		return allAccessPoints, errors.WithStackTrace(err)
	}

	input := &s3control.ListAccessPointsInput{
		AccountId: aws.String(accountId),
	}

	err = svc.ListAccessPointsPages(
		input,
		func(page *s3control.ListAccessPointsOutput, lastPage bool) bool {
			for _, accessPoint := range page.AccessPointList {
				if shouldIncludeS3AccessPoint(accessPoint, configObj) {
					allAccessPoints = append(allAccessPoints, accessPoint.Name)
				}
			}
			return !lastPage
		},
	)
	return allAccessPoints, errors.WithStackTrace(err)
}

func shouldIncludeS3AccessPoint(accessPoint *s3control.AccessPoint, configObj config.Config) bool {
	if accessPoint == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(accessPoint.Name),
		configObj.S3AccessPoint.IncludeRule.NamesRegExp,
		configObj.S3AccessPoint.ExcludeRule.NamesRegExp,
	)
}

func nukeAllS3AccessPoints(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	svc := s3control.New(session)

	accountId, err := util.GetCurrentAccountId(session)
	if err != nil {
		logging.Logger.Errorf("Cannot get account id for nuking S3 Access Points")
		return CannotGetAccountIdErr{}
	}

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No S3 Access Points to nuke in region %s", region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on S3AccessPoint.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many S3 Access Points at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyS3AccessPointsErr{}
	}

	var deletedNames []*string

	logging.Logger.Debugf("Deleting S3 Access Points in region %s", region)
	for _, accessPoint := range identifiers {
		input := &s3control.DeleteAccessPointInput{
			AccountId: aws.String(accountId),
			Name:      accessPoint,
		}

		_, err := svc.DeleteAccessPoint(input)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(accessPoint),
			ResourceType: "S3 Access Point",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking S3 Access Point",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
			logging.Logger.Debugf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, accessPoint)
			logging.Logger.Debugf("Deleted S3 Access Point: %s", aws.StringValue(accessPoint))
		}
	}

	logging.Logger.Debugf("[OK] %d S3 Access Points deleted in %s", len(deletedNames), *session.Config.Region)

	return nil
}

func getAllS3ObjectLambdaAccessPoints(session *session.Session, configObj config.Config) ([]*string, error) {
	svc := s3control.New(session)

	allObjectLambdaAccessPoints := []*string{}

	accountId, err := util.GetCurrentAccountId(session)
	if err != nil {
		return allObjectLambdaAccessPoints, errors.WithStackTrace(err)
	}

	input := &s3control.ListAccessPointsForObjectLambdaInput{
		AccountId: aws.String(accountId),
	}

	err = svc.ListAccessPointsForObjectLambdaPages(
		input,
		func(page *s3control.ListAccessPointsForObjectLambdaOutput, lastPage bool) bool {
			for _, objectLambdaAccessPoint := range page.ObjectLambdaAccessPointList {
				if shouldIncludeS3ObjectLambdaAccessPoint(objectLambdaAccessPoint, configObj) {
					allObjectLambdaAccessPoints = append(allObjectLambdaAccessPoints, objectLambdaAccessPoint.Name)
				}
			}
			return !lastPage
		},
	)
	return allObjectLambdaAccessPoints, errors.WithStackTrace(err)
}

func shouldIncludeS3ObjectLambdaAccessPoint(accessPoint *s3control.ObjectLambdaAccessPoint, configObj config.Config) bool {
	if accessPoint == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(accessPoint.Name),
		configObj.S3ObjectLambdaAccessPoint.IncludeRule.NamesRegExp,
		configObj.S3ObjectLambdaAccessPoint.ExcludeRule.NamesRegExp,
	)
}

func nukeAllS3ObjectLambdaAccessPoints(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	svc := s3control.New(session)

	accountId, err := util.GetCurrentAccountId(session)
	if err != nil {
		logging.Logger.Errorf("Cannot get account id for nuking S3 Object Lambda Access Points")
		return CannotGetAccountIdErr{}
	}

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No S3 Object Lambda Access Points to nuke in region %s", region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on S3ObjectLambdaAccessPoint.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many S3 Object Lambda Access Points at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyS3ObjectLambdaAccessPointsErr{}
	}

	var deletedNames []*string

	logging.Logger.Debugf("Deleting S3 Object Lambda Access Points in region %s", region)
	for _, accessPoint := range identifiers {
		input := &s3control.DeleteAccessPointForObjectLambdaInput{
			AccountId: aws.String(accountId),
			Name:      accessPoint,
		}

		_, err := svc.DeleteAccessPointForObjectLambda(input)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(accessPoint),
			ResourceType: "S3 Object Lambda Access Point",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking S3 Object Lambda Access Point",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
			logging.Logger.Debugf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, accessPoint)
			logging.Logger.Debugf("Deleted S3 Object Lambda Access Point: %s", aws.StringValue(accessPoint))
		}
	}

	logging.Logger.Debugf("[OK] %d S3 Object Lambda Access Points deleted in %s", len(deletedNames), *session.Config.Region)

	return nil
}

func getAllS3MultiRegionAccessPoints(session *session.Session, configObj config.Config) ([]*string, error) {
	// NOTE: this action will always be routed to the US West (Oregon) Region.
	svc := s3control.New(session, &aws.Config{Region: aws.String("us-west-2")})

	allMultiRegionAccessPoint := []*string{}

	accountId, err := util.GetCurrentAccountId(session)
	if err != nil {
		return allMultiRegionAccessPoint, errors.WithStackTrace(err)
	}

	input := &s3control.ListMultiRegionAccessPointsInput{
		AccountId: aws.String(accountId),
	}

	err = svc.ListMultiRegionAccessPointsPages(
		input,
		func(page *s3control.ListMultiRegionAccessPointsOutput, lastPage bool) bool {
			for _, multiRegionAccessPoint := range page.AccessPoints {
				if shouldIncludeS3MultiRegionAccessPoint(multiRegionAccessPoint, configObj) {
					allMultiRegionAccessPoint = append(allMultiRegionAccessPoint, multiRegionAccessPoint.Name)
				}
			}
			return !lastPage
		},
	)
	return allMultiRegionAccessPoint, errors.WithStackTrace(err)
}

func shouldIncludeS3MultiRegionAccessPoint(accessPoint *s3control.MultiRegionAccessPointReport, configObj config.Config) bool {
	if accessPoint == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(accessPoint.Name),
		configObj.S3MultiRegionAccessPoint.IncludeRule.NamesRegExp,
		configObj.S3MultiRegionAccessPoint.ExcludeRule.NamesRegExp,
	)
}

func nukeAllS3MultiRegionAccessPoints(session *session.Session, identifiers []*string) error {
	// NOTE: this action will always be routed to the US West (Oregon) Region.
	svc := s3control.New(session, &aws.Config{Region: aws.String("us-west-2")})

	accountId, err := util.GetCurrentAccountId(session)
	if err != nil {
		logging.Logger.Errorf("Cannot get account id for nuking S3 Multi Region Access Points")
		return CannotGetAccountIdErr{}
	}

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No S3 Multi Region Access Points to nuke")
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on S3MultiRegionAccessPoint.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many S3 Multi Region Access Points at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyS3MultiRegionAccessPointsErr{}
	}

	logging.Logger.Debugf("Deleting S3 Multi Region Access Points in region")
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, s3Mrap := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteS3MultiRegionAccessPointAsync(wg, errChans[i], svc, s3Mrap, accountId)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking S3 Multi Region Access Point",
			}, map[string]interface{}{})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	// Now wait until the S3 multi region access point are deleted
	err = retry.DoWithRetry(
		logging.Logger,
		"Waiting for all S3 multi region access points to be deleted.",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			areDeleted, err := areAllS3MultiRegionAccessPointsDeleted(svc, identifiers, accountId)
			if err != nil {
				return errors.WithStackTrace(retry.FatalError{Underlying: err})
			}
			if areDeleted {
				return nil
			}
			return fmt.Errorf("Not all S3 multi region access points deleted.")
		},
	)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	for _, s3Mrap := range identifiers {
		logging.Logger.Debugf("[OK] S3 Multi Region Access Point %s was deleted", aws.StringValue(s3Mrap))
	}
	return nil
}

// areAllS3MultiRegionAccessPointsDeleted returns true
func areAllS3MultiRegionAccessPointsDeleted(svc *s3control.S3Control, identifiers []*string, accountId string) (bool, error) {
	for _, identifier := range identifiers {
		resp, err := svc.GetMultiRegionAccessPoint(&s3control.GetMultiRegionAccessPointInput{
			AccountId: aws.String(accountId),
			Name:      identifier,
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchMultiRegionAccessPoint" { // s3 multi region access point deleted
				continue
			} else {
				return false, err
			}
		}

		if resp != nil {
			return false, nil
		}
	}

	// At this point, all the S3 multi region access points are either nil, or deleted.
	return true, nil
}

// deleteS3MultiRegionAccessPointAsync deletes the provided S3 Multi Region Access Point asynchronously ina goroutine,
// using wait groups for concurrency control and a return channel for errors.
func deleteS3MultiRegionAccessPointAsync(wg *sync.WaitGroup, errChan chan error, svc *s3control.S3Control, s3Mrap *string, accountId string) {
	defer wg.Done()

	input := &s3control.DeleteMultiRegionAccessPointInput{
		AccountId: aws.String(accountId),
		Details: &s3control.DeleteMultiRegionAccessPointInput_{
			Name: s3Mrap,
		},
	}
	_, err := svc.DeleteMultiRegionAccessPoint(input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(s3Mrap),
		ResourceType: "S3 Multi Region Access Point",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}

// Custom errors

type CannotGetAccountIdErr struct{}

type TooManyS3AccessPointsErr struct{}

type TooManyS3ObjectLambdaAccessPointsErr struct{}

type TooManyS3MultiRegionAccessPointsErr struct{}

func (err CannotGetAccountIdErr) Error() string {
	return "Cannot get account id for nuking S3 Access Points"
}

func (err TooManyS3AccessPointsErr) Error() string {
	return "Too many S3 Access Points requested at once."
}

func (err TooManyS3ObjectLambdaAccessPointsErr) Error() string {
	return "Too many S3 Object Lambda Access Points requested at once."
}

func (err TooManyS3MultiRegionAccessPointsErr) Error() string {
	return "Too many S3 Multi Region Access Points requested at once."
}

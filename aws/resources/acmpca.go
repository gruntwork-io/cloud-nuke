package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// GetAll returns a list of all arn's of ACMPCA, which can be deleted.
func (ap *ACMPCA) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var arns []*string

	paginator := acmpca.NewListCertificateAuthoritiesPaginator(ap.Client, &acmpca.ListCertificateAuthoritiesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, ca := range page.CertificateAuthorities {
			if ap.shouldInclude(ca, configObj) {
				arns = append(arns, ca.Arn)
			}
		}
	}

	return arns, nil
}

func (ap *ACMPCA) shouldInclude(ca types.CertificateAuthority, configObj config.Config) bool {
	statusSafe := ca.Status
	if statusSafe == types.CertificateAuthorityStatusDeleted {
		return false
	}

	// reference time for excludeAfter is lastStateChangeAt time,
	// unless it was never changed and createAt time is used.
	var referenceTime time.Time
	if ca.LastStateChangeAt == nil {
		referenceTime = aws.ToTime(ca.CreatedAt)
	} else {
		referenceTime = aws.ToTime(ca.LastStateChangeAt)
	}

	return configObj.ACMPCA.ShouldInclude(config.ResourceValue{Time: &referenceTime})
}

// nukeAll will delete all ACMPCA, which are given by a list of arn's.
func (ap *ACMPCA) nukeAll(arns []*string) error {
	if len(arns) == 0 {
		logging.Debugf("No ACMPCA to nuke in region %s", ap.Region)
		return nil
	}

	logging.Debugf("Deleting all ACMPCA in region %s", ap.Region)
	// There is no bulk delete acmpca API, so we delete the batch of ARNs concurrently using go routines.
	wg := new(sync.WaitGroup)
	wg.Add(len(arns))
	errChans := make([]chan error, len(arns))
	for i, arn := range arns {
		errChans[i] = make(chan error, 1)
		go ap.deleteAsync(wg, errChans[i], arn)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Errorf("[Failed] %s", err)
		}
	}

	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

// deleteAsync deletes the provided ACMPCA arn. Intended to be run in a goroutine, using wait groups
// and a return channel for errors.
func (ap *ACMPCA) deleteAsync(wg *sync.WaitGroup, errChan chan error, arn *string) {
	defer wg.Done()

	logging.Debugf("Fetching details of CA to be deleted for ACMPCA %s in region %s", *arn, ap.Region)
	details, detailsErr := ap.Client.DescribeCertificateAuthority(
		ap.Context,
		&acmpca.DescribeCertificateAuthorityInput{CertificateAuthorityArn: arn})
	if detailsErr != nil {
		errChan <- detailsErr
		return
	}
	if details.CertificateAuthority == nil {
		errChan <- fmt.Errorf("could not find CA %s", aws.ToString(arn))
		return
	}
	if details.CertificateAuthority.Status == "" {
		errChan <- fmt.Errorf("could not fetch status for CA %s", aws.ToString(arn))
		return
	}

	// find out, whether we have to disable the CA first, prior to deletion.
	statusSafe := details.CertificateAuthority.Status
	shouldUpdateStatus := statusSafe != types.CertificateAuthorityStatusCreating &&
		statusSafe != types.CertificateAuthorityStatusPendingCertificate &&
		statusSafe != types.CertificateAuthorityStatusDisabled &&
		statusSafe != types.CertificateAuthorityStatusDeleted

	if shouldUpdateStatus {
		logging.Debugf("Setting status to 'DISABLED' for ACMPCA %s in region %s", *arn, ap.Region)
		if _, updateStatusErr := ap.Client.UpdateCertificateAuthority(ap.Context, &acmpca.UpdateCertificateAuthorityInput{
			CertificateAuthorityArn: arn,
			Status:                  types.CertificateAuthorityStatusDisabled,
		}); updateStatusErr != nil {
			errChan <- updateStatusErr
			return
		}

		logging.Debugf("Did set status to 'DISABLED' for ACMPCA: %s in region %s", *arn, ap.Region)
	}

	_, deleteErr := ap.Client.DeleteCertificateAuthority(ap.Context, &acmpca.DeleteCertificateAuthorityInput{
		CertificateAuthorityArn: arn,
		// the range is 7 to 30 days.
		// since cloud-nuke should not be used in production,
		// we assume that the minimum (7 days) is fine.
		PermanentDeletionTimeInDays: aws.Int32(7),
	})

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.ToString(arn),
		ResourceType: "ACM Private CA (ACMPCA)",
		Error:        deleteErr,
	}
	report.Record(e)

	if deleteErr != nil {
		errChan <- deleteErr
		return
	}
	logging.Debugf("Deleted ACMPCA: %s successfully", *arn)
	errChan <- nil
}

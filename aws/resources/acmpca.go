package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// GetAll returns a list of all arns of ACMPCA, which can be deleted.
func (ap *ACMPCA) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var arns []*string
	paginationErr := ap.Client.ListCertificateAuthoritiesPages(
		&acmpca.ListCertificateAuthoritiesInput{},
		func(p *acmpca.ListCertificateAuthoritiesOutput, lastPage bool) bool {
			for _, ca := range p.CertificateAuthorities {
				if ap.shouldInclude(ca, configObj) {
					arns = append(arns, ca.Arn)
				}
			}
			return !lastPage
		})
	if paginationErr != nil {
		return nil, errors.WithStackTrace(paginationErr)
	}

	return arns, nil
}

func (ap *ACMPCA) shouldInclude(ca *acmpca.CertificateAuthority, configObj config.Config) bool {
	if ca == nil {
		return false
	}

	statusSafe := aws.StringValue(ca.Status)
	if statusSafe == acmpca.CertificateAuthorityStatusDeleted {
		return false
	}

	// reference time for excludeAfter is lastStateChangeAt time,
	// unless it was never changed and createAt time is used.
	var referenceTime time.Time
	if ca.LastStateChangeAt == nil {
		referenceTime = aws.TimeValue(ca.CreatedAt)
	} else {
		referenceTime = aws.TimeValue(ca.LastStateChangeAt)
	}

	return configObj.ACMPCA.ShouldInclude(config.ResourceValue{Time: &referenceTime})
}

// nukeAll will delete all ACMPCA, which are given by a list of arns.
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
		&acmpca.DescribeCertificateAuthorityInput{CertificateAuthorityArn: arn})
	if detailsErr != nil {
		errChan <- detailsErr
		return
	}
	if details.CertificateAuthority == nil {
		errChan <- fmt.Errorf("could not find CA %s", aws.StringValue(arn))
		return
	}
	if details.CertificateAuthority.Status == nil {
		errChan <- fmt.Errorf("could not fetch status for CA %s", aws.StringValue(arn))
		return
	}

	// find out, whether we have to disable the CA first, prior to deletion.
	statusSafe := aws.StringValue(details.CertificateAuthority.Status)
	shouldUpdateStatus := statusSafe != acmpca.CertificateAuthorityStatusCreating &&
		statusSafe != acmpca.CertificateAuthorityStatusPendingCertificate &&
		statusSafe != acmpca.CertificateAuthorityStatusDisabled &&
		statusSafe != acmpca.CertificateAuthorityStatusDeleted

	if shouldUpdateStatus {
		logging.Debugf("Setting status to 'DISABLED' for ACMPCA %s in region %s", *arn, ap.Region)
		if _, updateStatusErr := ap.Client.UpdateCertificateAuthority(&acmpca.UpdateCertificateAuthorityInput{
			CertificateAuthorityArn: arn,
			Status:                  aws.String(acmpca.CertificateAuthorityStatusDisabled),
		}); updateStatusErr != nil {
			errChan <- updateStatusErr
			return
		}

		logging.Debugf("Did set status to 'DISABLED' for ACMPCA: %s in region %s", *arn, ap.Region)
	}

	_, deleteErr := ap.Client.DeleteCertificateAuthority(&acmpca.DeleteCertificateAuthorityInput{
		CertificateAuthorityArn: arn,
		// the range is 7 to 30 days.
		// since cloud-nuke should not be used in production,
		// we assume that the minimum (7 days) is fine.
		PermanentDeletionTimeInDays: aws.Int64(7),
	})

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(arn),
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

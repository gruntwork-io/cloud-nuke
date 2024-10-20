package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a list of strings of ACM ARNs
func (a *ACM) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var acmArns []*string
	paginator := acm.NewListCertificatesPaginator(a.Client, &acm.ListCertificatesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cert := range page.CertificateSummaryList {
			logging.Debug(fmt.Sprintf("Found ACM %s with domain name %s", *cert.CertificateArn, *cert.DomainName))
			if a.shouldInclude(cert, configObj) {
				logging.Debug(fmt.Sprintf("Including ACM %s", *cert.CertificateArn))
				acmArns = append(acmArns, cert.CertificateArn)
			} else {
				logging.Debug(fmt.Sprintf("Skipping ACM %s", *cert.CertificateArn))
			}
		}
	}

	return acmArns, nil
}

func (a *ACM) shouldInclude(acm types.CertificateSummary, configObj config.Config) bool {
	if acm.InUse != nil && *acm.InUse {
		logging.Debug(fmt.Sprintf("ACM %s is in use", *acm.CertificateArn))
		return false
	}

	shouldInclude := configObj.ACM.ShouldInclude(config.ResourceValue{
		Name: acm.DomainName,
		Time: acm.CreatedAt,
	})
	logging.Debug(fmt.Sprintf("shouldInclude result for ACM: %s w/ domain name: %s, time: %s, and config: %+v",
		*acm.CertificateArn, *acm.DomainName, acm.CreatedAt, configObj.ACM))
	return shouldInclude
}

// Deletes all ACMs
func (a *ACM) nukeAll(arns []*string) error {
	if len(arns) == 0 {
		logging.Debugf("No ACMs to nuke in region %s", a.Region)
		return nil
	}

	logging.Debugf("Deleting all ACMs in region %s", a.Region)
	deletedCount := 0
	for _, acmArn := range arns {
		params := &acm.DeleteCertificateInput{
			CertificateArn: acmArn,
		}

		_, err := a.Client.DeleteCertificate(a.Context, params)
		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedCount++
			logging.Debugf("Deleted ACM: %s", *acmArn)
		}

		e := report.Entry{
			Identifier:   *acmArn,
			ResourceType: "ACM",
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d ACM(s) terminated in %s", deletedCount, a.Region)
	return nil
}

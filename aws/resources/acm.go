package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/pterm/pterm"
)

// Returns a list of strings of ACM ARNs
func (a *ACM) getAll(c context.Context, configObj config.Config) ([]*string, error) {

	params := &acm.ListCertificatesInput{}

	acmArns := []*string{}
	err := a.Client.ListCertificatesPages(params,
		func(page *acm.ListCertificatesOutput, lastPage bool) bool {
			for i := range page.CertificateSummaryList {
				pterm.Debug.Println(fmt.Sprintf("Found ACM %s with domain name %s",
					*page.CertificateSummaryList[i].CertificateArn, *page.CertificateSummaryList[i].DomainName))
				if a.shouldInclude(page.CertificateSummaryList[i], configObj) {
					pterm.Debug.Println(fmt.Sprintf(
						"Including ACM %s", *page.CertificateSummaryList[i].CertificateArn))
					acmArns = append(acmArns, page.CertificateSummaryList[i].CertificateArn)
				} else {
					pterm.Debug.Println(fmt.Sprintf(
						"Skipping ACM %s", *page.CertificateSummaryList[i].CertificateArn))
				}
			}

			return !lastPage
		},
	)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return acmArns, nil
}

func (a *ACM) shouldInclude(acm *acm.CertificateSummary, configObj config.Config) bool {
	if acm == nil {
		return false
	}

	if acm.InUse != nil && *acm.InUse {
		pterm.Debug.Println(fmt.Sprintf("ACM %s is in use", *acm.CertificateArn))
		return false
	}

	return configObj.ACM.ShouldInclude(config.ResourceValue{
		Name: acm.DomainName,
		Time: acm.CreatedAt,
	})
}

// Deletes all ACMs
func (a *ACM) nukeAll(arns []*string) error {
	if len(arns) == 0 {
		logging.Logger.Debugf("No ACMs to nuke in region %s", a.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all ACMs in region %s", a.Region)

	deletedCount := 0
	for _, acmArn := range arns {
		params := &acm.DeleteCertificateInput{
			CertificateArn: acmArn,
		}

		_, err := a.Client.DeleteCertificate(params)
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ACM",
			}, map[string]interface{}{
				"region": a.Region,
			})
		} else {
			deletedCount++
			logging.Logger.Debugf("Deleted ACM: %s", *acmArn)
		}

		e := report.Entry{
			Identifier:   *acmArn,
			ResourceType: "ACM",
			Error:        err,
		}
		report.Record(e)
	}

	logging.Logger.Debugf("[OK] %d ACM(s) terminated in %s", deletedCount, a.Region)
	return nil
}

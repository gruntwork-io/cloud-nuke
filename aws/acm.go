package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

// Returns a list of strings of ACM ARNs
func getAllACMs(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
	svc := acm.New(session)

	params := &acm.ListCertificatesInput{}

	acmArns := []string{}
	err := svc.ListCertificatesPages(params,
		func(page *acm.ListCertificatesOutput, lastPage bool) bool {
			for i := range page.CertificateSummaryList {
				if shouldIncludeACM(page.CertificateSummaryList[i], excludeAfter, configObj) {
					acmArns = append(acmArns, *page.CertificateSummaryList[i].CertificateArn)
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

func shouldIncludeACM(acm *acm.CertificateSummary, excludeAfter time.Time, configObj config.Config) bool {
	if acm == nil {
		return false
	}

	if acm.InUse != nil && *acm.InUse {
		logging.Logger.Debugf("Skipping ACM %s as it is in use", *acm.CertificateArn)
		return false
	}

	if acm.CreatedAt != nil {
		if excludeAfter.Before(*acm.CreatedAt) {
			return false
		}
	}

	return config.ShouldInclude(
		*acm.DomainName,
		configObj.ACM.IncludeRule.NamesRegExp,
		configObj.ACM.ExcludeRule.NamesRegExp,
	)
}

// Deletes all ACMs
func nukeAllACMs(session *session.Session, acmArns []*string) error {
	svc := acm.New(session)

	if len(acmArns) == 0 {
		logging.Logger.Debugf("No ACMs to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all ACMs in region %s", *session.Config.Region)

	deletedCount := 0
	for _, acmArn := range acmArns {
		params := &acm.DeleteCertificateInput{
			CertificateArn: acmArn,
		}

		_, err := svc.DeleteCertificate(params)
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ACM",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedCount++
			logging.Logger.Debugf("Deleted ACM: %s", *acmArn)
		}
	}

	logging.Logger.Debugf("[OK] %d ACM(s) terminated in %s", deletedCount, *session.Config.Region)

	return nil
}

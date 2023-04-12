package aws

import (
	goerror "errors"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAllMacieMemberAccounts will find and return any Macie accounts that were created via accepting an invite from another AWS Account
// Unfortunately, the Macie API doesn't provide the metadata information we'd need to implement the excludeAfter or configObj patterns, so we
// currently can only accept a session
func getAllMacieMemberAccounts(session *session.Session) ([]string, error) {
	svc := macie2.New(session)
	stssvc := sts.New(session)

	allMacieAccounts := []string{}
	output, err := svc.GetAdministratorAccount(&macie2.GetAdministratorAccountInput{})
	if err != nil {
		// There are several different errors that AWS may return when you attempt to call Macie operations on an account
		// that doesn't yet have Macie enabled. For our purposes, this is fine, as we're only looking for those accounts and
		// regions where Macie is enabled. Therefore, we ignore only these expected errors, and return any other error that might occur
		var ade *macie2.AccessDeniedException
		var rnfe *macie2.ResourceNotFoundException

		switch {
		case goerror.As(err, &ade):
			logging.Logger.Debugf("Macie AccessDeniedException means macie is not enabled in account, so skipping")
			return allMacieAccounts, nil
		case goerror.As(err, &rnfe):
			logging.Logger.Debugf("Macie ResourceNotFoundException means macie is not enabled in account, so skipping")
			return allMacieAccounts, nil
		default:
			return allMacieAccounts, errors.WithStackTrace(err)
		}
	}
	// If the current account does have an Administrator account relationship, and it is enabled, then we consider this a macie member account
	if output.Administrator != nil && output.Administrator.RelationshipStatus != nil {
		if aws.StringValue(output.Administrator.RelationshipStatus) == macie2.RelationshipStatusEnabled {

			input := &sts.GetCallerIdentityInput{}
			output, err := stssvc.GetCallerIdentity(input)
			if err != nil {
				return allMacieAccounts, errors.WithStackTrace(err)
			}

			currentAccountId := aws.StringValue(output.Account)

			allMacieAccounts = append(allMacieAccounts, currentAccountId)
		}
	}

	return allMacieAccounts, nil
}

func nukeAllMacieMemberAccounts(session *session.Session, identifiers []string) error {
	svc := macie2.New(session)
	region := aws.StringValue(session.Config.Region)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No Macie member accounts to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting Macie account membership and disabling Macie in %s", region)

	for _, accountId := range identifiers {
		_, disassociateErr := svc.DisassociateFromAdministratorAccount(&macie2.DisassociateFromAdministratorAccountInput{})

		if disassociateErr != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking MACIE",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
			return errors.WithStackTrace(disassociateErr)
		}

		_, err := svc.DisableMacie(&macie2.DisableMacieInput{})

		// Record status of this resource
		e := report.Entry{
			Identifier:   accountId,
			ResourceType: "Macie member account",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking MACIE",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
			return errors.WithStackTrace(err)
		}

		logging.Logger.Debugf("[OK] Macie account association for accountId %s deleted in %s", accountId, region)
	}

	return nil
}

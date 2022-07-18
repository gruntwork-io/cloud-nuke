package aws

import (
	goerror "errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllMacieMemberAccounts(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
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
		logging.Logger.Infof("No Macie member accounts to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting Macie account membership and disabling Macie in %s", region)

	for _, accountId := range identifiers {
		_, disassociateErr := svc.DisassociateFromAdministratorAccount(&macie2.DisassociateFromAdministratorAccountInput{})

		if disassociateErr != nil {
			return errors.WithStackTrace(disassociateErr)
		}

		_, err := svc.DisableMacie(&macie2.DisableMacieInput{})
		if err != nil {
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("[OK] Macie account association for accountId %s deleted in %s", accountId, region)
	}

	return nil
}

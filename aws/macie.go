package aws

import (
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// GetMacieSession will find and return any Macie accounts that were created via accepting an invite from another AWS Account
// Unfortunately, the Macie API doesn't provide the metadata information we'd need to implement the configObj pattern, so we
// currently can only accept a session and excludeAfter

func getMacieStatus(session *session.Session, excludeAfter time.Time) ([]string, error) {
	svc := macie2.New(session)
	var macieStatus []string

	output, err := svc.GetMacieSession(&macie2.GetMacieSessionInput{})

	if err != nil {
		// If Macie is not enabled when we call GetMacieSession, we get back an error
		// so we should ignore the error if it's just telling us the account/region is not
		// enabled and return nil to indicate there are no resources to nuke
		if strings.Contains(err.Error(), "Macie is not enabled") {
			return nil, nil
		}
		return nil, errors.WithStackTrace(err)
	}

	if shouldIncludeMacie(output, excludeAfter) {
		macieStatus = append(macieStatus, *output.Status)
	}

	return macieStatus, nil
}

func shouldIncludeMacie(macie *macie2.GetMacieSessionOutput, excludeAfter time.Time) bool {
	if excludeAfter.Before(*macie.UpdatedAt) {
		return false
	}
	return true
}

func getAllMacieMembers(svc *macie2.Macie2) ([]*string, error) {
	var memberAccountIds []*string

	// OnlyAssociated=false input parameter includes "pending" invite members
	members, err := svc.ListMembers(&macie2.ListMembersInput{OnlyAssociated: aws.String("false")})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	for _, member := range members.Members {
		memberAccountIds = append(memberAccountIds, member.AccountId)
	}

	for awsgo.StringValue(members.NextToken) != "" {
		members, err = svc.ListMembers(&macie2.ListMembersInput{NextToken: members.NextToken})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, member := range members.Members {
			memberAccountIds = append(memberAccountIds, member.AccountId)
		}
	}
	logging.Logger.Debugf("Found %d member accounts attached to macie", len(memberAccountIds))
	return memberAccountIds, nil
}

func removeMacieMembers(svc *macie2.Macie2, accountIds []*string) {

	// Member accounts must first be disassociated
	for _, accountId := range accountIds {
		_, err := svc.DisassociateMember(&macie2.DisassociateMemberInput{Id: accountId})
		if err != nil {
			if strings.Contains(err.Error(), "there isn't a delegated Macie administrator") {
				logging.Logger.Infof("No delegated Macie administrator found.")
			} else {
				logging.Logger.Errorf("[Failed] Failed to disassociate members from macie")
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error disassociating members from macie",
				}, map[string]interface{}{
					"region": *svc.Config.Region,
					"reason": "Unable to disassociate",
				})
			}
			return
		}
		logging.Logger.Debugf("%s member account disassociated", accountId)

		// Once disassociated, member accounts can be deleted
		_, err = svc.DeleteMember(&macie2.DeleteMemberInput{Id: accountId})
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed to delete members from macie")
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error deleting members from macie",
			}, map[string]interface{}{
				"region": *svc.Config.Region,
				"reason": "Unable to delete",
			})
			return
		}
		logging.Logger.Debugf("%s member account deleted", accountId)
	}
}

func disassociateMacieAdministratorAccount(svc *macie2.Macie2) {

	// Check for an associated admin account as it must be disassociated before disabling macie
	adminAccount, err := svc.GetAdministratorAccount(&macie2.GetAdministratorAccountInput{})
	if err != nil {
		logging.Logger.Errorf("[Failed] Failed to check for administrator account")
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error checking for administrator account in macie",
		}, map[string]interface{}{
			"region": *svc.Config.Region,
			"reason": "Unable to find admin account",
		})
	}
	// Disassociate if one exists
	if adminAccount.Administrator != nil {
		_, err := svc.DisassociateFromAdministratorAccount(&macie2.DisassociateFromAdministratorAccountInput{})
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed to disassociate from administrator account")
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error disassociating administrator account in macie",
			}, map[string]interface{}{
				"region": *svc.Config.Region,
				"reason": "Unable to disassociate admin account",
			})
		} else {
			logging.Logger.Debugf("Successfully disassociated from administrator account")
		}
	}
}

func nukeAllMacieMemberAccounts(session *session.Session, identifier []string) error {
	svc := macie2.New(session)
	region := aws.StringValue(session.Config.Region)

	if len(identifier) == 0 {
		logging.Logger.Debugf("No Macie member accounts to nuke in region %s", region)
		return nil
	}

	// Check for and remove any member accounts in Macie
	// Macie cannot be disabled with active member accounts
	memberAccountIds, err := getAllMacieMembers(svc)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error finding macie member accounts",
		}, map[string]interface{}{
			"region": *svc.Config.Region,
			"reason": "Error finding macie member accounts",
		})
	}
	if err == nil && len(memberAccountIds) > 0 {
		removeMacieMembers(svc, memberAccountIds)
	}

	// Disassociate account from any administrator accounts
	// Macie cannot be disable with an active administrator account
	disassociateMacieAdministratorAccount(svc)

	// Disable Macie
	_, err = svc.DisableMacie(&macie2.DisableMacieInput{})
	if err != nil {
		logging.Logger.Errorf("[Failed] Failed to disable macie.")
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error disabling macie",
		}, map[string]interface{}{
			"region": *svc.Config.Region,
			"reason": "Error disabling macie",
		})
		e := report.Entry{
			Identifier:   aws.StringValue(&identifier[0]),
			ResourceType: "Macie",
			Error:        err,
		}
		report.Record(e)
	} else {
		logging.Logger.Debugf("[OK] Macie %s disabled", identifier[0])
		e := report.Entry{
			Identifier:   aws.StringValue(&identifier[0]),
			ResourceType: "Macie",
		}
		report.Record(e)
	}
	return nil
}

// func nukeAllMacieMemberAccounts(session *session.Session, identifiers []string) error {
// 	svc := macie2.New(session)
// 	region := aws.StringValue(session.Config.Region)

// 	if len(identifiers) == 0 {
// 		logging.Logger.Debugf("No Macie member accounts to nuke in region %s", *session.Config.Region)
// 		return nil
// 	}

// 	logging.Logger.Debugf("Deleting Macie account membership and disabling Macie in %s", region)

// 	for _, accountId := range identifiers {
// 		_, disassociateErr := svc.DisassociateFromAdministratorAccount(&macie2.DisassociateFromAdministratorAccountInput{})

// 		if disassociateErr != nil {
// 			telemetry.TrackEvent(commonTelemetry.EventContext{
// 				EventName: "Error Nuking MACIE",
// 			}, map[string]interface{}{
// 				"region": *session.Config.Region,
// 			})
// 			return errors.WithStackTrace(disassociateErr)
// 		}

// 		_, err := svc.DisableMacie(&macie2.DisableMacieInput{})

// 		// Record status of this resource
// 		e := report.Entry{
// 			Identifier:   accountId,
// 			ResourceType: "Macie member account",
// 			Error:        err,
// 		}
// 		report.Record(e)

// 		if err != nil {
// 			telemetry.TrackEvent(commonTelemetry.EventContext{
// 				EventName: "Error Nuking MACIE",
// 			}, map[string]interface{}{
// 				"region": *session.Config.Region,
// 			})
// 			return errors.WithStackTrace(err)
// 		}

// 		logging.Logger.Debugf("[OK] Macie account association for accountId %s deleted in %s", accountId, region)
// 	}

// 	return nil
// }

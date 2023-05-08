package aws

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func getAllSecurityHubArns(session *session.Session, excludeAfter time.Time) ([]string, error) {
	svc := securityhub.New(session)
	var securityHubArns []string

	output, err := svc.DescribeHub(&securityhub.DescribeHubInput{})

	if err != nil {
		// If Security Hub is not enabled when we call DescribeHub, we get back an error
		// so we should ignore the error if it's just telling us the account/region is not
		// subscribed and return nil to indicate there are no resources to nuke
		if strings.Contains(err.Error(), "is not subscribed to AWS Security Hub") {
			return nil, nil
		}
		return nil, errors.WithStackTrace(err)
	}

	if shouldIncludeHub(output, excludeAfter) {
		securityHubArns = append(securityHubArns, *output.HubArn)
	}

	return securityHubArns, nil
}

func shouldIncludeHub(hub *securityhub.DescribeHubOutput, excludeAfter time.Time) bool {
	subscribedAt, err := time.Parse(time.RFC3339, *hub.SubscribedAt)
	if err != nil {
		logging.Logger.Debugf("Could not parse subscribedAt timestamp (%s) of security hub. Excluding from delete.", *hub.SubscribedAt)
		return false
	}
	if excludeAfter.Before(subscribedAt) {
		return false
	}
	return true
}

func getAllSecurityHubMembers(svc *securityhub.SecurityHub) ([]*string, error) {
	var hubMemberAccountIds []*string

	// OnlyAssociated=false input parameter includes "pending" invite members
	members, err := svc.ListMembers(&securityhub.ListMembersInput{OnlyAssociated: aws.Bool(false)})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	for _, member := range members.Members {
		hubMemberAccountIds = append(hubMemberAccountIds, member.AccountId)
	}

	for awsgo.StringValue(members.NextToken) != "" {
		members, err = svc.ListMembers(&securityhub.ListMembersInput{NextToken: members.NextToken})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, member := range members.Members {
			hubMemberAccountIds = append(hubMemberAccountIds, member.AccountId)
		}
	}
	logging.Logger.Debugf("Found %d member accounts attached to security hub", len(hubMemberAccountIds))
	return hubMemberAccountIds, nil
}

func removeMembersFromHub(svc *securityhub.SecurityHub, accountIds []*string) error {

	// Member accounts must first be disassociated
	_, err := svc.DisassociateMembers(&securityhub.DisassociateMembersInput{AccountIds: accountIds})
	if err != nil {
		return err
	}
	logging.Logger.Debugf("%d member accounts disassociated", len(accountIds))

	// Once disassociated, member accounts can be deleted
	_, err = svc.DeleteMembers(&securityhub.DeleteMembersInput{AccountIds: accountIds})
	if err != nil {
		return err
	}
	logging.Logger.Debugf("%d member accounts deleted", len(accountIds))

	return nil
}

func nukeSecurityHub(session *session.Session, securityHubArns []string) error {
	svc := securityhub.New(session)

	if len(securityHubArns) == 0 {
		logging.Logger.Debugf("No security hub resources to nuke in region %s", *session.Config.Region)
		return nil
	}

	// Check for any member accounts in security hub
	// Security Hub cannot be disabled with active member accounts
	memberAccountIds, err := getAllSecurityHubMembers(svc)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error finding security hub member accounts",
		}, map[string]interface{}{
			"region": *svc.Config.Region,
			"reason": "Error finding security hub member accounts",
		})
	}

	// Remove any member accounts if they exist
	if err == nil && len(memberAccountIds) > 0 {
		err = removeMembersFromHub(svc, memberAccountIds)
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed to disassociate members from security hub")
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error disassociating members from security hub",
			}, map[string]interface{}{
				"region": *svc.Config.Region,
				"reason": "Unable to disassociate",
			})
		}
	}

	// Check for an administrator account
	// Security hub cannot be disabled with an active administrator account
	adminAccount, err := svc.GetAdministratorAccount(&securityhub.GetAdministratorAccountInput{})
	if err != nil {
		logging.Logger.Errorf("[Failed] Failed to check for administrator account")
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error checking for administrator account in security hub",
		}, map[string]interface{}{
			"region": *svc.Config.Region,
			"reason": "Unable to find admin account",
		})
	}

	// Disassociate administrator account if it exists
	if adminAccount.Administrator != nil {
		_, err := svc.DisassociateFromAdministratorAccount(&securityhub.DisassociateFromAdministratorAccountInput{})
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed to disassociate from administrator account")
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error disassociating administrator account in security hub",
			}, map[string]interface{}{
				"region": *svc.Config.Region,
				"reason": "Unable to disassociate admin account",
			})
		}
	}

	// Disable security hub
	_, err = svc.DisableSecurityHub(&securityhub.DisableSecurityHubInput{})
	if err != nil {
		logging.Logger.Errorf("[Failed] Failed to disable security hub.")
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error disabling security hub",
		}, map[string]interface{}{
			"region": *svc.Config.Region,
			"reason": "Error disabling security hub",
		})
		e := report.Entry{
			Identifier:   aws.StringValue(&securityHubArns[0]),
			ResourceType: "Security Hub",
			Error:        err,
		}
		report.Record(e)
	} else {
		logging.Logger.Debugf("[OK] Security Hub %s disabled", securityHubArns[0])
		e := report.Entry{
			Identifier:   aws.StringValue(&securityHubArns[0]),
			ResourceType: "Security Hub",
		}
		report.Record(e)
	}
	return nil
}

package resources

import (
	"github.com/gruntwork-io/cloud-nuke/config"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (sh *SecurityHub) getAll(configObj config.Config) ([]*string, error) {
	var securityHubArns []*string

	output, err := sh.Client.DescribeHub(&securityhub.DescribeHubInput{})

	if err != nil {
		// If Security Hub is not enabled when we call DescribeHub, we get back an error
		// so we should ignore the error if it's just telling us the account/region is not
		// subscribed and return nil to indicate there are no resources to nuke
		if strings.Contains(err.Error(), "is not subscribed to AWS Security Hub") {
			return nil, nil
		}
		return nil, errors.WithStackTrace(err)
	}

	if shouldIncludeHub(output, configObj) {
		securityHubArns = append(securityHubArns, output.HubArn)
	}

	return securityHubArns, nil
}

func shouldIncludeHub(hub *securityhub.DescribeHubOutput, configObj config.Config) bool {
	subscribedAt, err := time.Parse(time.RFC3339, *hub.SubscribedAt)
	if err != nil {
		logging.Logger.Debugf(
			"Could not parse subscribedAt timestamp (%s) of security hub. Excluding from delete.", *hub.SubscribedAt)
		return false
	}

	return configObj.SecurityHub.ShouldInclude(config.ResourceValue{Time: &subscribedAt})
}

func (sh *SecurityHub) getAllSecurityHubMembers() ([]*string, error) {
	var hubMemberAccountIds []*string

	// OnlyAssociated=false input parameter includes "pending" invite members
	members, err := sh.Client.ListMembers(&securityhub.ListMembersInput{OnlyAssociated: aws.Bool(false)})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	for _, member := range members.Members {
		hubMemberAccountIds = append(hubMemberAccountIds, member.AccountId)
	}

	for awsgo.StringValue(members.NextToken) != "" {
		members, err = sh.Client.ListMembers(&securityhub.ListMembersInput{NextToken: members.NextToken})
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

func (sh *SecurityHub) removeMembersFromHub(accountIds []*string) error {

	// Member accounts must first be disassociated
	_, err := sh.Client.DisassociateMembers(&securityhub.DisassociateMembersInput{AccountIds: accountIds})
	if err != nil {
		return err
	}
	logging.Logger.Debugf("%d member accounts disassociated", len(accountIds))

	// Once disassociated, member accounts can be deleted
	_, err = sh.Client.DeleteMembers(&securityhub.DeleteMembersInput{AccountIds: accountIds})
	if err != nil {
		return err
	}
	logging.Logger.Debugf("%d member accounts deleted", len(accountIds))

	return nil
}

func (sh *SecurityHub) nukeAll(securityHubArns []string) error {
	if len(securityHubArns) == 0 {
		logging.Logger.Debugf("No security hub resources to nuke in region %s", sh.Region)
		return nil
	}

	// Check for any member accounts in security hub
	// Security Hub cannot be disabled with active member accounts
	memberAccountIds, err := sh.getAllSecurityHubMembers()
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error finding security hub member accounts",
		}, map[string]interface{}{
			"region": sh.Region,
			"reason": "Error finding security hub member accounts",
		})
	}

	// Remove any member accounts if they exist
	if err == nil && len(memberAccountIds) > 0 {
		err = sh.removeMembersFromHub(memberAccountIds)
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed to disassociate members from security hub")
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error disassociating members from security hub",
			}, map[string]interface{}{
				"region": sh.Region,
				"reason": "Unable to disassociate",
			})
		}
	}

	// Check for an administrator account
	// Security hub cannot be disabled with an active administrator account
	adminAccount, err := sh.Client.GetAdministratorAccount(&securityhub.GetAdministratorAccountInput{})
	if err != nil {
		logging.Logger.Errorf("[Failed] Failed to check for administrator account")
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error checking for administrator account in security hub",
		}, map[string]interface{}{
			"region": sh.Region,
			"reason": "Unable to find admin account",
		})
	}

	// Disassociate administrator account if it exists
	if adminAccount.Administrator != nil {
		_, err := sh.Client.DisassociateFromAdministratorAccount(&securityhub.DisassociateFromAdministratorAccountInput{})
		if err != nil {
			logging.Logger.Errorf("[Failed] Failed to disassociate from administrator account")
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error disassociating administrator account in security hub",
			}, map[string]interface{}{
				"region": sh.Region,
				"reason": "Unable to disassociate admin account",
			})
		}
	}

	// Disable security hub
	_, err = sh.Client.DisableSecurityHub(&securityhub.DisableSecurityHubInput{})
	if err != nil {
		logging.Logger.Errorf("[Failed] Failed to disable security hub.")
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error disabling security hub",
		}, map[string]interface{}{
			"region": sh.Region,
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

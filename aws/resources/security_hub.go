package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (sh *SecurityHub) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var securityHubArns []*string

	output, err := sh.Client.DescribeHub(sh.Context, &securityhub.DescribeHubInput{})

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
	subscribedAt, err := util.ParseTimestamp(hub.SubscribedAt)
	if err != nil {
		logging.Debugf(
			"Could not parse subscribedAt timestamp (%s) of security hub. Excluding from delete.", *hub.SubscribedAt)
		return false
	}

	return configObj.SecurityHub.ShouldInclude(config.ResourceValue{Time: subscribedAt})
}

func (sh *SecurityHub) getAllSecurityHubMembers() ([]*string, error) {
	var hubMemberAccountIds []*string

	// OnlyAssociated=false input parameter includes "pending" invite members
	members, err := sh.Client.ListMembers(sh.Context, &securityhub.ListMembersInput{OnlyAssociated: aws.Bool(false)})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	for _, member := range members.Members {
		hubMemberAccountIds = append(hubMemberAccountIds, member.AccountId)
	}

	for aws.ToString(members.NextToken) != "" {
		members, err = sh.Client.ListMembers(sh.Context, &securityhub.ListMembersInput{NextToken: members.NextToken})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, member := range members.Members {
			hubMemberAccountIds = append(hubMemberAccountIds, member.AccountId)
		}
	}
	logging.Debugf("Found %d member accounts attached to security hub", len(hubMemberAccountIds))
	return hubMemberAccountIds, nil
}

func (sh *SecurityHub) removeMembersFromHub(accountIds []*string) error {

	// Member accounts must first be disassociated
	_, err := sh.Client.DisassociateMembers(sh.Context, &securityhub.DisassociateMembersInput{AccountIds: aws.ToStringSlice(accountIds)})
	if err != nil {
		return err
	}
	logging.Debugf("%d member accounts disassociated", len(accountIds))

	// Once disassociated, member accounts can be deleted
	_, err = sh.Client.DeleteMembers(sh.Context, &securityhub.DeleteMembersInput{AccountIds: aws.ToStringSlice(accountIds)})
	if err != nil {
		return err
	}
	logging.Debugf("%d member accounts deleted", len(accountIds))

	return nil
}

func (sh *SecurityHub) nukeAll(securityHubArns []string) error {
	if len(securityHubArns) == 0 {
		logging.Debugf("No security hub resources to nuke in region %s", sh.Region)
		return nil
	}

	// Check for any member accounts in security hub
	// Security Hub cannot be disabled with active member accounts
	memberAccountIds, err := sh.getAllSecurityHubMembers()
	if err != nil {
		return err
	}

	// Remove any member accounts if they exist
	if len(memberAccountIds) > 0 {
		err = sh.removeMembersFromHub(memberAccountIds)
		if err != nil {
			logging.Errorf("[Failed] Failed to disassociate members from security hub")
		}
	}

	// Check for an administrator account
	// Security hub cannot be disabled with an active administrator account
	adminAccount, err := sh.Client.GetAdministratorAccount(sh.Context, &securityhub.GetAdministratorAccountInput{})
	if err != nil {
		logging.Errorf("[Failed] Failed to check for administrator account")
	}

	// Disassociate administrator account if it exists
	if adminAccount.Administrator != nil {
		_, err := sh.Client.DisassociateFromAdministratorAccount(sh.Context, &securityhub.DisassociateFromAdministratorAccountInput{})
		if err != nil {
			logging.Errorf("[Failed] Failed to disassociate from administrator account")
		}
	}

	// Disable security hub
	_, err = sh.Client.DisableSecurityHub(sh.Context, &securityhub.DisableSecurityHubInput{})
	if err != nil {
		logging.Errorf("[Failed] Failed to disable security hub.")
		e := report.Entry{
			Identifier:   aws.ToString(&securityHubArns[0]),
			ResourceType: "Security Hub",
			Error:        err,
		}
		report.Record(e)
	} else {
		logging.Debugf("[OK] Security Hub %s disabled", securityHubArns[0])
		e := report.Entry{
			Identifier:   aws.ToString(&securityHubArns[0]),
			ResourceType: "Security Hub",
		}
		report.Record(e)
	}
	return nil
}

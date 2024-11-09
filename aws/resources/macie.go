package resources

import (
	"context"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// getAll will find and return any Macie accounts that were created via accepting an invite from another AWS Account
// Unfortunately, the Macie API doesn't provide the metadata information we'd need to implement the configObj pattern, so we
// currently can only accept a session and excludeAfter
func (mm *MacieMember) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var macieStatus []*string

	output, err := mm.Client.GetMacieSession(mm.Context, &macie2.GetMacieSessionInput{})
	if err != nil {
		// If Macie is not enabled when we call GetMacieSession, we get back an error
		// so we should ignore the error if it's just telling us the account/region is not
		// enabled and return nil to indicate there are no resources to nuke
		if strings.Contains(err.Error(), "Macie is not enabled") {
			return nil, nil
		}
		return nil, errors.WithStackTrace(err)
	}

	// Note: there's no identifier for the Macie resource, so we just insert random elements to the return array
	//  to follow the similar framework as other resources.
	if configObj.MacieMember.ShouldInclude(config.ResourceValue{
		Time: output.CreatedAt,
	}) {
		macieStatus = append(macieStatus, aws.String(string(output.Status)))
	}

	return macieStatus, nil
}

func (mm *MacieMember) getAllMacieMembers() ([]*string, error) {
	var memberAccountIds []*string

	// OnlyAssociated=false input parameter includes "pending" invite members
	members, err := mm.Client.ListMembers(mm.Context, &macie2.ListMembersInput{OnlyAssociated: aws.String("false")})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	for _, member := range members.Members {
		memberAccountIds = append(memberAccountIds, member.AccountId)
	}

	for aws.ToString(members.NextToken) != "" {
		members, err = mm.Client.ListMembers(mm.Context, &macie2.ListMembersInput{NextToken: members.NextToken})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, member := range members.Members {
			memberAccountIds = append(memberAccountIds, member.AccountId)
		}
	}
	logging.Debugf("Found %d member accounts attached to macie", len(memberAccountIds))
	return memberAccountIds, nil
}

func (mm *MacieMember) removeMacieMembers(memberAccountIds []*string) error {

	// Member accounts must first be disassociated
	for _, accountId := range memberAccountIds {
		_, err := mm.Client.DisassociateMember(mm.Context, &macie2.DisassociateMemberInput{Id: accountId})
		if err != nil {
			return err
		}
		logging.Debugf("%s member account disassociated", *accountId)

		// Once disassociated, member accounts can be deleted
		_, err = mm.Client.DeleteMember(mm.Context, &macie2.DeleteMemberInput{Id: accountId})
		if err != nil {
			return err
		}
		logging.Debugf("%s member account deleted", *accountId)
	}
	return nil
}

func (mm *MacieMember) nukeAll(identifier []string) error {
	if len(identifier) == 0 {
		logging.Debugf("No Macie member accounts to nuke in region %s", mm.Region)
		return nil
	}

	// Check for and remove any member accounts in Macie
	// Macie cannot be disabled with active member accounts
	memberAccountIds, err := mm.getAllMacieMembers()
	if err != nil {
	}
	if err == nil && len(memberAccountIds) > 0 {
		err = mm.removeMacieMembers(memberAccountIds)
		if err != nil {
			logging.Errorf("[Failed] Failed to remove members from macie")
		}
	}

	// Check for an administrator account
	// Macie cannot be disabled with an active administrator account
	adminAccount, err := mm.Client.GetAdministratorAccount(mm.Context, &macie2.GetAdministratorAccountInput{})
	if err != nil {
		if strings.Contains(err.Error(), "there isn't a delegated Macie administrator") {
			logging.Debugf("No delegated Macie administrator found to remove.")
		} else {
			logging.Errorf("[Failed] Failed to check for administrator account")
		}
	}

	// Disassociate administrator account if it exists
	if adminAccount.Administrator != nil {
		_, err := mm.Client.DisassociateFromAdministratorAccount(mm.Context, &macie2.DisassociateFromAdministratorAccountInput{})
		if err != nil {
			logging.Errorf("[Failed] Failed to disassociate from administrator account")
		}
	}

	// Disable Macie
	_, err = mm.Client.DisableMacie(mm.Context, &macie2.DisableMacieInput{})
	if err != nil {
		logging.Errorf("[Failed] Failed to disable macie.")
		e := report.Entry{
			Identifier:   aws.ToString(&identifier[0]),
			ResourceType: "Macie",
			Error:        err,
		}
		report.Record(e)
	} else {
		logging.Debugf("[OK] Macie disabled in %s", mm.Region)
		e := report.Entry{
			Identifier:   mm.Region,
			ResourceType: "Macie",
		}
		report.Record(e)
	}
	return nil
}

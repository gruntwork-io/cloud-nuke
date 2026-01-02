package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// MacieMemberAPI defines the interface for Macie operations.
type MacieMemberAPI interface {
	GetMacieSession(ctx context.Context, params *macie2.GetMacieSessionInput, optFns ...func(*macie2.Options)) (*macie2.GetMacieSessionOutput, error)
	ListMembers(ctx context.Context, params *macie2.ListMembersInput, optFns ...func(*macie2.Options)) (*macie2.ListMembersOutput, error)
	DisassociateMember(ctx context.Context, params *macie2.DisassociateMemberInput, optFns ...func(*macie2.Options)) (*macie2.DisassociateMemberOutput, error)
	DeleteMember(ctx context.Context, params *macie2.DeleteMemberInput, optFns ...func(*macie2.Options)) (*macie2.DeleteMemberOutput, error)
	GetAdministratorAccount(ctx context.Context, params *macie2.GetAdministratorAccountInput, optFns ...func(*macie2.Options)) (*macie2.GetAdministratorAccountOutput, error)
	DisassociateFromAdministratorAccount(ctx context.Context, params *macie2.DisassociateFromAdministratorAccountInput, optFns ...func(*macie2.Options)) (*macie2.DisassociateFromAdministratorAccountOutput, error)
	DisableMacie(ctx context.Context, params *macie2.DisableMacieInput, optFns ...func(*macie2.Options)) (*macie2.DisableMacieOutput, error)
}

// NewMacieMember creates a new MacieMember resource using the generic resource pattern.
func NewMacieMember() AwsResource {
	return NewAwsResource(&resource.Resource[MacieMemberAPI]{
		ResourceTypeName: "macie-member",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[MacieMemberAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = macie2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.MacieMember
		},
		Lister: listMacieSessions,
		Nuker:  deleteMacieSessions,
	})
}

// listMacieSessions retrieves Macie sessions that match the config filters.
// Unlike most resources, Macie has a single session per region, so we return
// the status as an identifier if the session exists and matches filters.
func listMacieSessions(ctx context.Context, client MacieMemberAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.GetMacieSession(ctx, &macie2.GetMacieSessionInput{})
	if err != nil {
		// If Macie is not enabled, the API returns an error indicating so.
		// This is expected behavior, not an error - just return nil to indicate
		// there are no resources to nuke.
		if strings.Contains(err.Error(), "Macie is not enabled") {
			return nil, nil
		}
		return nil, errors.WithStackTrace(err)
	}

	// Use status as identifier since Macie doesn't have a unique resource ID
	if cfg.ShouldInclude(config.ResourceValue{Time: output.CreatedAt}) {
		return []*string{aws.String(string(output.Status))}, nil
	}

	return nil, nil
}

// getAllMacieMembers retrieves all member accounts attached to Macie with pagination.
func getAllMacieMembers(ctx context.Context, client MacieMemberAPI) ([]*string, error) {
	var memberAccountIds []*string

	paginator := macie2.NewListMembersPaginator(client, &macie2.ListMembersInput{
		OnlyAssociated: aws.String("false"), // Include "pending" invite members
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		for _, member := range page.Members {
			memberAccountIds = append(memberAccountIds, member.AccountId)
		}
	}

	logging.Debugf("Found %d member accounts attached to Macie", len(memberAccountIds))
	return memberAccountIds, nil
}

// removeMacieMembers disassociates and deletes all member accounts.
// Members must be disassociated before they can be deleted.
func removeMacieMembers(ctx context.Context, client MacieMemberAPI, memberAccountIds []*string) error {
	for _, accountId := range memberAccountIds {
		if accountId == nil {
			continue
		}

		// Disassociate the member first
		if _, err := client.DisassociateMember(ctx, &macie2.DisassociateMemberInput{Id: accountId}); err != nil {
			return err
		}
		logging.Debugf("Disassociated Macie member account: %s", *accountId)

		// Then delete the member
		if _, err := client.DeleteMember(ctx, &macie2.DeleteMemberInput{Id: accountId}); err != nil {
			return err
		}
		logging.Debugf("Deleted Macie member account: %s", *accountId)
	}
	return nil
}

// deleteMacieSessions is a custom nuker for Macie resources.
// Macie requires special handling because:
// 1. Member accounts must be disassociated and deleted before disabling
// 2. Administrator account must be disassociated before disabling
// 3. The service itself is disabled (not deleted like typical resources)
func deleteMacieSessions(ctx context.Context, client MacieMemberAPI, scope resource.Scope, resourceType string, identifiers []*string) []resource.NukeResult {
	if len(identifiers) == 0 {
		logging.Debugf("No Macie resources to nuke in %s", scope)
		return nil
	}

	logging.Infof("Deleting %d %s in %s", len(identifiers), resourceType, scope)

	// Step 1: Remove member accounts (Macie cannot be disabled with active members)
	memberAccountIds, err := getAllMacieMembers(ctx, client)
	if err != nil {
		logging.Debugf("Failed to get Macie members: %s", err)
	}

	if len(memberAccountIds) > 0 {
		if err := removeMacieMembers(ctx, client, memberAccountIds); err != nil {
			logging.Debugf("Failed to remove Macie members: %s", err)
		}
	}

	// Step 2: Disassociate from administrator account if one exists
	adminAccount, err := client.GetAdministratorAccount(ctx, &macie2.GetAdministratorAccountInput{})
	if err != nil {
		if !strings.Contains(err.Error(), "there isn't a delegated Macie administrator") {
			logging.Debugf("Failed to check for administrator account: %s", err)
		}
	}

	if adminAccount != nil && adminAccount.Administrator != nil {
		if _, err := client.DisassociateFromAdministratorAccount(ctx, &macie2.DisassociateFromAdministratorAccountInput{}); err != nil {
			logging.Debugf("Failed to disassociate from administrator account: %s", err)
		}
	}

	// Step 3: Disable Macie
	_, err = client.DisableMacie(ctx, &macie2.DisableMacieInput{})

	return []resource.NukeResult{{Identifier: aws.ToString(identifiers[0]), Error: err}}
}

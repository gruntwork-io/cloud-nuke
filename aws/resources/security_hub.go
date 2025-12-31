package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// SecurityHubAPI defines the interface for Security Hub operations.
type SecurityHubAPI interface {
	DescribeHub(ctx context.Context, params *securityhub.DescribeHubInput, optFns ...func(*securityhub.Options)) (*securityhub.DescribeHubOutput, error)
	ListMembers(ctx context.Context, params *securityhub.ListMembersInput, optFns ...func(*securityhub.Options)) (*securityhub.ListMembersOutput, error)
	DisassociateMembers(ctx context.Context, params *securityhub.DisassociateMembersInput, optFns ...func(*securityhub.Options)) (*securityhub.DisassociateMembersOutput, error)
	DeleteMembers(ctx context.Context, params *securityhub.DeleteMembersInput, optFns ...func(*securityhub.Options)) (*securityhub.DeleteMembersOutput, error)
	GetAdministratorAccount(ctx context.Context, params *securityhub.GetAdministratorAccountInput, optFns ...func(*securityhub.Options)) (*securityhub.GetAdministratorAccountOutput, error)
	DisassociateFromAdministratorAccount(ctx context.Context, params *securityhub.DisassociateFromAdministratorAccountInput, optFns ...func(*securityhub.Options)) (*securityhub.DisassociateFromAdministratorAccountOutput, error)
	DisableSecurityHub(ctx context.Context, params *securityhub.DisableSecurityHubInput, optFns ...func(*securityhub.Options)) (*securityhub.DisableSecurityHubOutput, error)
}

// NewSecurityHub creates a new SecurityHub resource using the generic resource pattern.
func NewSecurityHub() AwsResource {
	return NewAwsResource(&resource.Resource[SecurityHubAPI]{
		ResourceTypeName: "security-hub",
		BatchSize:        5,
		InitClient: func(r *resource.Resource[SecurityHubAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for SecurityHub client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = securityhub.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SecurityHub
		},
		Lister: listSecurityHubs,
		Nuker:  deleteSecurityHubs,
	})
}

// listSecurityHubs retrieves all Security Hubs that match the config filters.
func listSecurityHubs(ctx context.Context, client SecurityHubAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var securityHubArns []*string

	output, err := client.DescribeHub(ctx, &securityhub.DescribeHubInput{})

	if err != nil {
		// If Security Hub is not enabled when we call DescribeHub, we get back an error
		// so we should ignore the error if it's just telling us the account/region is not
		// subscribed and return nil to indicate there are no resources to nuke
		if strings.Contains(err.Error(), "is not subscribed to AWS Security Hub") {
			return nil, nil
		}
		return nil, errors.WithStackTrace(err)
	}

	if shouldIncludeSecurityHub(output, cfg) {
		securityHubArns = append(securityHubArns, output.HubArn)
	}

	return securityHubArns, nil
}

func shouldIncludeSecurityHub(hub *securityhub.DescribeHubOutput, cfg config.ResourceType) bool {
	subscribedAt, err := util.ParseTimestamp(hub.SubscribedAt)
	if err != nil {
		logging.Debugf(
			"Could not parse subscribedAt timestamp (%s) of security hub. Excluding from delete.", *hub.SubscribedAt)
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{Time: subscribedAt})
}

func getAllSecurityHubMembers(ctx context.Context, client SecurityHubAPI) ([]*string, error) {
	var hubMemberAccountIds []*string

	// OnlyAssociated=false input parameter includes "pending" invite members
	members, err := client.ListMembers(ctx, &securityhub.ListMembersInput{OnlyAssociated: aws.Bool(false)})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	for _, member := range members.Members {
		hubMemberAccountIds = append(hubMemberAccountIds, member.AccountId)
	}

	for aws.ToString(members.NextToken) != "" {
		members, err = client.ListMembers(ctx, &securityhub.ListMembersInput{NextToken: members.NextToken})
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

func removeMembersFromSecurityHub(ctx context.Context, client SecurityHubAPI, accountIds []*string) error {
	// Member accounts must first be disassociated
	_, err := client.DisassociateMembers(ctx, &securityhub.DisassociateMembersInput{AccountIds: aws.ToStringSlice(accountIds)})
	if err != nil {
		return err
	}
	logging.Debugf("%d member accounts disassociated", len(accountIds))

	// Once disassociated, member accounts can be deleted
	_, err = client.DeleteMembers(ctx, &securityhub.DeleteMembersInput{AccountIds: aws.ToStringSlice(accountIds)})
	if err != nil {
		return err
	}
	logging.Debugf("%d member accounts deleted", len(accountIds))

	return nil
}

// deleteSecurityHubs is a custom nuker for Security Hub resources.
func deleteSecurityHubs(ctx context.Context, client SecurityHubAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No security hub resources to nuke in %s", scope)
		return nil
	}

	// Check for any member accounts in security hub
	// Security Hub cannot be disabled with active member accounts
	memberAccountIds, err := getAllSecurityHubMembers(ctx, client)
	if err != nil {
		return err
	}

	// Remove any member accounts if they exist
	if len(memberAccountIds) > 0 {
		err = removeMembersFromSecurityHub(ctx, client, memberAccountIds)
		if err != nil {
			logging.Errorf("[Failed] Failed to disassociate members from security hub")
		}
	}

	// Check for an administrator account
	// Security hub cannot be disabled with an active administrator account
	adminAccount, err := client.GetAdministratorAccount(ctx, &securityhub.GetAdministratorAccountInput{})
	if err != nil {
		logging.Errorf("[Failed] Failed to check for administrator account")
	}

	// Disassociate administrator account if it exists
	if adminAccount.Administrator != nil {
		_, err = client.DisassociateFromAdministratorAccount(ctx, &securityhub.DisassociateFromAdministratorAccountInput{})
		if err != nil {
			logging.Errorf("[Failed] Failed to disassociate from administrator account")
		}
	}

	// Disable security hub
	_, err = client.DisableSecurityHub(ctx, &securityhub.DisableSecurityHubInput{})
	if err != nil {
		logging.Errorf("[Failed] Failed to disable security hub.")
		e := report.Entry{
			Identifier:   aws.ToString(identifiers[0]),
			ResourceType: resourceType,
			Error:        err,
		}
		report.Record(e)
	} else {
		logging.Debugf("[OK] Security Hub %s disabled", aws.ToString(identifiers[0]))
		e := report.Entry{
			Identifier:   aws.ToString(identifiers[0]),
			ResourceType: resourceType,
		}
		report.Record(e)
	}
	return nil
}

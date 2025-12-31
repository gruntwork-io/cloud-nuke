package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SecurityHubAPI interface {
	DescribeHub(ctx context.Context, params *securityhub.DescribeHubInput, optFns ...func(*securityhub.Options)) (*securityhub.DescribeHubOutput, error)
	ListMembers(ctx context.Context, params *securityhub.ListMembersInput, optFns ...func(*securityhub.Options)) (*securityhub.ListMembersOutput, error)
	DisassociateMembers(ctx context.Context, params *securityhub.DisassociateMembersInput, optFns ...func(*securityhub.Options)) (*securityhub.DisassociateMembersOutput, error)
	DeleteMembers(ctx context.Context, params *securityhub.DeleteMembersInput, optFns ...func(*securityhub.Options)) (*securityhub.DeleteMembersOutput, error)
	GetAdministratorAccount(ctx context.Context, params *securityhub.GetAdministratorAccountInput, optFns ...func(*securityhub.Options)) (*securityhub.GetAdministratorAccountOutput, error)
	DisassociateFromAdministratorAccount(ctx context.Context, params *securityhub.DisassociateFromAdministratorAccountInput, optFns ...func(*securityhub.Options)) (*securityhub.DisassociateFromAdministratorAccountOutput, error)
	DisableSecurityHub(ctx context.Context, params *securityhub.DisableSecurityHubInput, optFns ...func(*securityhub.Options)) (*securityhub.DisableSecurityHubOutput, error)
}

type SecurityHub struct {
	BaseAwsResource
	Client  SecurityHubAPI
	Region  string
	HubArns []string
}

func (sh *SecurityHub) Init(cfg aws.Config) {
	sh.Client = securityhub.NewFromConfig(cfg)
}

func (sh *SecurityHub) ResourceName() string {
	return "security-hub"
}

func (sh *SecurityHub) ResourceIdentifiers() []string {
	return sh.HubArns
}

func (sh *SecurityHub) MaxBatchSize() int {
	return 5
}

func (sh *SecurityHub) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SecurityHub
}

func (sh *SecurityHub) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sh.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sh.HubArns = aws.ToStringSlice(identifiers)
	return sh.HubArns, nil
}

func (sh *SecurityHub) Nuke(ctx context.Context, identifiers []string) error {
	if err := sh.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type MacieMemberAPI interface {
	GetMacieSession(ctx context.Context, params *macie2.GetMacieSessionInput, optFns ...func(*macie2.Options)) (*macie2.GetMacieSessionOutput, error)
	ListMembers(ctx context.Context, params *macie2.ListMembersInput, optFns ...func(*macie2.Options)) (*macie2.ListMembersOutput, error)
	DisassociateMember(ctx context.Context, params *macie2.DisassociateMemberInput, optFns ...func(*macie2.Options)) (*macie2.DisassociateMemberOutput, error)
	DeleteMember(ctx context.Context, params *macie2.DeleteMemberInput, optFns ...func(*macie2.Options)) (*macie2.DeleteMemberOutput, error)
	GetAdministratorAccount(ctx context.Context, params *macie2.GetAdministratorAccountInput, optFns ...func(*macie2.Options)) (*macie2.GetAdministratorAccountOutput, error)
	DisassociateFromAdministratorAccount(ctx context.Context, params *macie2.DisassociateFromAdministratorAccountInput, optFns ...func(*macie2.Options)) (*macie2.DisassociateFromAdministratorAccountOutput, error)
	DisableMacie(ctx context.Context, params *macie2.DisableMacieInput, optFns ...func(*macie2.Options)) (*macie2.DisableMacieOutput, error)
}

type MacieMember struct {
	BaseAwsResource
	Client     MacieMemberAPI
	Region     string
	AccountIds []string
}

func (mm *MacieMember) InitV2(cfg aws.Config) {
	mm.Client = macie2.NewFromConfig(cfg)
}

func (mm *MacieMember) IsUsingV2() bool { return true }

func (mm *MacieMember) ResourceName() string {
	return "macie-member"
}

func (mm *MacieMember) ResourceIdentifiers() []string {
	return mm.AccountIds
}

func (mm *MacieMember) MaxBatchSize() int {
	return 10
}

func (mm *MacieMember) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.MacieMember
}

func (mm *MacieMember) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := mm.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	mm.AccountIds = aws.ToStringSlice(identifiers)
	return mm.AccountIds, nil
}

func (mm *MacieMember) Nuke(identifiers []string) error {
	if err := mm.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

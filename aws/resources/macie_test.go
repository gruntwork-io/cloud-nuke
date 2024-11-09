package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/aws/aws-sdk-go-v2/service/macie2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedMacie struct {
	MacieMemberAPI
	GetMacieSessionOutput                      macie2.GetMacieSessionOutput
	ListMembersOutput                          macie2.ListMembersOutput
	DisassociateMemberOutput                   macie2.DisassociateMemberOutput
	DeleteMemberOutput                         macie2.DeleteMemberOutput
	GetAdministratorAccountOutput              macie2.GetAdministratorAccountOutput
	DisassociateFromAdministratorAccountOutput macie2.DisassociateFromAdministratorAccountOutput
	DisableMacieOutput                         macie2.DisableMacieOutput
}

func (m mockedMacie) GetMacieSession(ctx context.Context, params *macie2.GetMacieSessionInput, optFns ...func(*macie2.Options)) (*macie2.GetMacieSessionOutput, error) {
	return &m.GetMacieSessionOutput, nil
}

func (m mockedMacie) ListMembers(ctx context.Context, params *macie2.ListMembersInput, optFns ...func(*macie2.Options)) (*macie2.ListMembersOutput, error) {
	return &m.ListMembersOutput, nil
}

func (m mockedMacie) DisassociateMember(ctx context.Context, params *macie2.DisassociateMemberInput, optFns ...func(*macie2.Options)) (*macie2.DisassociateMemberOutput, error) {
	return &m.DisassociateMemberOutput, nil
}

func (m mockedMacie) DeleteMember(ctx context.Context, params *macie2.DeleteMemberInput, optFns ...func(*macie2.Options)) (*macie2.DeleteMemberOutput, error) {
	return &m.DeleteMemberOutput, nil
}

func (m mockedMacie) GetAdministratorAccount(ctx context.Context, params *macie2.GetAdministratorAccountInput, optFns ...func(*macie2.Options)) (*macie2.GetAdministratorAccountOutput, error) {
	return &m.GetAdministratorAccountOutput, nil
}

func (m mockedMacie) DisassociateFromAdministratorAccount(ctx context.Context, params *macie2.DisassociateFromAdministratorAccountInput, optFns ...func(*macie2.Options)) (*macie2.DisassociateFromAdministratorAccountOutput, error) {
	return &m.DisassociateFromAdministratorAccountOutput, nil
}

func (m mockedMacie) DisableMacie(ctx context.Context, params *macie2.DisableMacieInput, optFns ...func(*macie2.Options)) (*macie2.DisableMacieOutput, error) {
	return &m.DisableMacieOutput, nil
}

func TestMacie_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mm := MacieMember{
		Client: mockedMacie{
			GetMacieSessionOutput: macie2.GetMacieSessionOutput{
				Status:    types.MacieStatusEnabled,
				CreatedAt: aws.Time(now),
			},
		},
	}

	_, err := mm.getAll(context.Background(), config.Config{})
	require.NoError(t, err)
}

func TestMacie_NukeAll(t *testing.T) {
	t.Parallel()

	mm := MacieMember{
		Client: mockedMacie{
			ListMembersOutput: macie2.ListMembersOutput{
				Members: []types.Member{
					{
						AccountId: aws.String("123456789012"),
					},
				},
			},

			DisassociateMemberOutput: macie2.DisassociateMemberOutput{},
			DeleteMemberOutput:       macie2.DeleteMemberOutput{},
			GetAdministratorAccountOutput: macie2.GetAdministratorAccountOutput{
				Administrator: &types.Invitation{
					AccountId: aws.String("123456789012"),
				},
			},
			DisassociateFromAdministratorAccountOutput: macie2.DisassociateFromAdministratorAccountOutput{},
			DisableMacieOutput:                         macie2.DisableMacieOutput{},
		},
	}

	err := mm.nukeAll([]string{"enabled"})
	require.NoError(t, err)
}

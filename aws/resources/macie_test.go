package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/aws/aws-sdk-go/service/macie2/macie2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedMacie struct {
	macie2iface.Macie2API
	GetMacieSessionOutput                      macie2.GetMacieSessionOutput
	ListMacieMembersOutput                     macie2.ListMembersOutput
	DisassociateMemberOutput                   macie2.DisassociateMemberOutput
	DeleteMemberOutput                         macie2.DeleteMemberOutput
	GetAdministratorAccountOutput              macie2.GetAdministratorAccountOutput
	DisassociateFromAdministratorAccountOutput macie2.DisassociateFromAdministratorAccountOutput
	DisableMacieOutput                         macie2.DisableMacieOutput
}

func (m mockedMacie) GetMacieSessionWithContext(_ aws.Context, input *macie2.GetMacieSessionInput, _ ...request.Option) (*macie2.GetMacieSessionOutput, error) {
	return &m.GetMacieSessionOutput, nil
}

func (m mockedMacie) ListMembersWithContext(_ aws.Context, _ *macie2.ListMembersInput, _ ...request.Option) (*macie2.ListMembersOutput, error) {
	return &m.ListMacieMembersOutput, nil
}

func (m mockedMacie) DisassociateMemberWithContext(_ aws.Context, _ *macie2.DisassociateMemberInput, _ ...request.Option) (*macie2.DisassociateMemberOutput, error) {
	return &m.DisassociateMemberOutput, nil
}

func (m mockedMacie) DeleteMemberWithContext(_ aws.Context, _ *macie2.DeleteMemberInput, _ ...request.Option) (*macie2.DeleteMemberOutput, error) {
	return &m.DeleteMemberOutput, nil
}

func (m mockedMacie) GetAdministratorAccountWithContext(_ aws.Context, _ *macie2.GetAdministratorAccountInput, _ ...request.Option) (*macie2.GetAdministratorAccountOutput, error) {
	return &m.GetAdministratorAccountOutput, nil
}

func (m mockedMacie) DisassociateFromAdministratorAccountWithContext(_ aws.Context, _ *macie2.DisassociateFromAdministratorAccountInput, _ ...request.Option) (*macie2.DisassociateFromAdministratorAccountOutput, error) {
	return &m.DisassociateFromAdministratorAccountOutput, nil
}

func (m mockedMacie) DisableMacieWithContext(_ aws.Context, _ *macie2.DisableMacieInput, _ ...request.Option) (*macie2.DisableMacieOutput, error) {
	return &m.DisableMacieOutput, nil
}

func TestMacie_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	mm := MacieMember{
		Client: mockedMacie{
			GetMacieSessionOutput: macie2.GetMacieSessionOutput{
				Status:    awsgo.String("ENABLED"),
				CreatedAt: awsgo.Time(now),
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
			ListMacieMembersOutput: macie2.ListMembersOutput{
				Members: []*macie2.Member{
					{
						AccountId: awsgo.String("123456789012"),
					},
				},
			},

			DisassociateMemberOutput: macie2.DisassociateMemberOutput{},
			DeleteMemberOutput:       macie2.DeleteMemberOutput{},
			GetAdministratorAccountOutput: macie2.GetAdministratorAccountOutput{
				Administrator: &macie2.Invitation{
					AccountId: awsgo.String("123456789012"),
				},
			},
			DisassociateFromAdministratorAccountOutput: macie2.DisassociateFromAdministratorAccountOutput{},
			DisableMacieOutput:                         macie2.DisableMacieOutput{},
		},
	}

	err := mm.nukeAll([]string{"enabled"})
	require.NoError(t, err)
}

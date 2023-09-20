package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/aws/aws-sdk-go/service/macie2/macie2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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

func (m mockedMacie) GetMacieSession(input *macie2.GetMacieSessionInput) (*macie2.GetMacieSessionOutput, error) {
	return &m.GetMacieSessionOutput, nil
}

func (m mockedMacie) ListMembers(input *macie2.ListMembersInput) (*macie2.ListMembersOutput, error) {
	return &m.ListMacieMembersOutput, nil
}

func (m mockedMacie) DisassociateMember(input *macie2.DisassociateMemberInput) (*macie2.DisassociateMemberOutput, error) {
	return &m.DisassociateMemberOutput, nil
}

func (m mockedMacie) DeleteMember(input *macie2.DeleteMemberInput) (*macie2.DeleteMemberOutput, error) {
	return &m.DeleteMemberOutput, nil
}

func (m mockedMacie) GetAdministratorAccount(input *macie2.GetAdministratorAccountInput) (*macie2.GetAdministratorAccountOutput, error) {
	return &m.GetAdministratorAccountOutput, nil
}

func (m mockedMacie) DisassociateFromAdministratorAccount(input *macie2.DisassociateFromAdministratorAccountInput) (*macie2.DisassociateFromAdministratorAccountOutput, error) {
	return &m.DisassociateFromAdministratorAccountOutput, nil
}

func (m mockedMacie) DisableMacie(input *macie2.DisableMacieInput) (*macie2.DisableMacieOutput, error) {
	return &m.DisableMacieOutput, nil
}

func TestMacie_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
	telemetry.InitTelemetry("cloud-nuke", "")
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

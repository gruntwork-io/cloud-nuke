package aws

import (
	"github.com/aws/aws-sdk-go/service/securityhub/securityhubiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
)

type mockedSecurityHub struct {
	securityhubiface.SecurityHubAPI
	DescribeHubOutput                          securityhub.DescribeHubOutput
	ListMembersOutput                          securityhub.ListMembersOutput
	DisassociateMembersOutput                  securityhub.DisassociateMembersOutput
	DeleteMembersOutput                        securityhub.DeleteMembersOutput
	GetAdministratorAccountOutput              securityhub.GetAdministratorAccountOutput
	DisassociateFromAdministratorAccountOutput securityhub.DisassociateFromAdministratorAccountOutput
	DisableSecurityHubOutput                   securityhub.DisableSecurityHubOutput
}

func (m mockedSecurityHub) DescribeHub(*securityhub.DescribeHubInput) (*securityhub.DescribeHubOutput, error) {
	return &m.DescribeHubOutput, nil
}

func (m mockedSecurityHub) ListMembers(*securityhub.ListMembersInput) (*securityhub.ListMembersOutput, error) {
	return &m.ListMembersOutput, nil
}

func (m mockedSecurityHub) DisassociateMembers(*securityhub.DisassociateMembersInput) (*securityhub.DisassociateMembersOutput, error) {
	return &m.DisassociateMembersOutput, nil
}

func (m mockedSecurityHub) DeleteMembers(*securityhub.DeleteMembersInput) (*securityhub.DeleteMembersOutput, error) {
	return &m.DeleteMembersOutput, nil
}

func (m mockedSecurityHub) GetAdministratorAccount(*securityhub.GetAdministratorAccountInput) (*securityhub.GetAdministratorAccountOutput, error) {
	return &m.GetAdministratorAccountOutput, nil
}

func (m mockedSecurityHub) DisassociateFromAdministratorAccount(*securityhub.DisassociateFromAdministratorAccountInput) (*securityhub.DisassociateFromAdministratorAccountOutput, error) {
	return &m.DisassociateFromAdministratorAccountOutput, nil
}

func (m mockedSecurityHub) DisableSecurityHub(*securityhub.DisableSecurityHubInput) (*securityhub.DisableSecurityHubOutput, error) {
	return &m.DisableSecurityHubOutput, nil
}

func TestSecurityHub_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	testArn := "test-arn"
	sh := SecurityHub{
		Client: mockedSecurityHub{
			DescribeHubOutput: securityhub.DescribeHubOutput{
				SubscribedAt: &nowStr,
				HubArn:       aws.String(testArn),
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := sh.getAll(config.Config{
				SecurityHub: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestSecurityHub_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	sh := SecurityHub{
		Client: mockedSecurityHub{
			ListMembersOutput: securityhub.ListMembersOutput{
				Members: []*securityhub.Member{{
					AccountId: aws.String("123456789012"),
				}},
			},
			DisassociateMembersOutput: securityhub.DisassociateMembersOutput{},
			DeleteMembersOutput:       securityhub.DeleteMembersOutput{},
			GetAdministratorAccountOutput: securityhub.GetAdministratorAccountOutput{
				Administrator: &securityhub.Invitation{
					AccountId: aws.String("123456789012"),
				},
			},
			DisassociateFromAdministratorAccountOutput: securityhub.DisassociateFromAdministratorAccountOutput{},
			DisableSecurityHubOutput:                   securityhub.DisableSecurityHubOutput{},
		},
	}

	err := sh.nukeAll([]string{"123456789012"})
	require.NoError(t, err)
}

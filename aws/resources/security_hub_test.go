package resources

import (
	"context"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/securityhub/securityhubiface"
	"github.com/gruntwork-io/cloud-nuke/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
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

func (m mockedSecurityHub) DescribeHubWithContext(_ awsgo.Context, _ *securityhub.DescribeHubInput, _ ...request.Option) (*securityhub.DescribeHubOutput, error) {
	return &m.DescribeHubOutput, nil
}

func (m mockedSecurityHub) ListMembersWithContext(_ awsgo.Context, _ *securityhub.ListMembersInput, _ ...request.Option) (*securityhub.ListMembersOutput, error) {
	return &m.ListMembersOutput, nil
}

func (m mockedSecurityHub) DisassociateMembersWithContext(_ awsgo.Context, _ *securityhub.DisassociateMembersInput, _ ...request.Option) (*securityhub.DisassociateMembersOutput, error) {
	return &m.DisassociateMembersOutput, nil
}

func (m mockedSecurityHub) DeleteMembersWithContext(_ awsgo.Context, _ *securityhub.DeleteMembersInput, _ ...request.Option) (*securityhub.DeleteMembersOutput, error) {
	return &m.DeleteMembersOutput, nil
}

func (m mockedSecurityHub) GetAdministratorAccountWithContext(_ awsgo.Context, _ *securityhub.GetAdministratorAccountInput, _ ...request.Option) (*securityhub.GetAdministratorAccountOutput, error) {
	return &m.GetAdministratorAccountOutput, nil
}

func (m mockedSecurityHub) DisassociateFromAdministratorAccountWithContext(_ awsgo.Context, _ *securityhub.DisassociateFromAdministratorAccountInput, _ ...request.Option) (*securityhub.DisassociateFromAdministratorAccountOutput, error) {
	return &m.DisassociateFromAdministratorAccountOutput, nil
}

func (m mockedSecurityHub) DisableSecurityHubWithContext(_ awsgo.Context, _ *securityhub.DisableSecurityHubInput, _ ...request.Option) (*securityhub.DisableSecurityHubOutput, error) {
	return &m.DisableSecurityHubOutput, nil
}

func TestSecurityHub_GetAll(t *testing.T) {

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
			names, err := sh.getAll(context.Background(), config.Config{
				SecurityHub: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestSecurityHub_NukeAll(t *testing.T) {

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

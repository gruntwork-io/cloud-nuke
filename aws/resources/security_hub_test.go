package resources

import (
	"context"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/stretchr/testify/require"
)

type mockedSecurityHub struct {
	SecurityHubAPI
	DescribeHubOutput                          securityhub.DescribeHubOutput
	ListMembersOutput                          securityhub.ListMembersOutput
	DisassociateMembersOutput                  securityhub.DisassociateMembersOutput
	DeleteMembersOutput                        securityhub.DeleteMembersOutput
	GetAdministratorAccountOutput              securityhub.GetAdministratorAccountOutput
	DisassociateFromAdministratorAccountOutput securityhub.DisassociateFromAdministratorAccountOutput
	DisableSecurityHubOutput                   securityhub.DisableSecurityHubOutput
}

func (m mockedSecurityHub) DescribeHub(_ context.Context, _ *securityhub.DescribeHubInput, _ ...func(*securityhub.Options)) (*securityhub.DescribeHubOutput, error) {
	return &m.DescribeHubOutput, nil
}

func (m mockedSecurityHub) ListMembers(_ context.Context, _ *securityhub.ListMembersInput, _ ...func(*securityhub.Options)) (*securityhub.ListMembersOutput, error) {
	return &m.ListMembersOutput, nil
}

func (m mockedSecurityHub) DisassociateMembers(_ context.Context, _ *securityhub.DisassociateMembersInput, _ ...func(*securityhub.Options)) (*securityhub.DisassociateMembersOutput, error) {
	return &m.DisassociateMembersOutput, nil
}

func (m mockedSecurityHub) DeleteMembers(_ context.Context, _ *securityhub.DeleteMembersInput, _ ...func(*securityhub.Options)) (*securityhub.DeleteMembersOutput, error) {
	return &m.DeleteMembersOutput, nil
}

func (m mockedSecurityHub) GetAdministratorAccount(_ context.Context, _ *securityhub.GetAdministratorAccountInput, _ ...func(*securityhub.Options)) (*securityhub.GetAdministratorAccountOutput, error) {
	return &m.GetAdministratorAccountOutput, nil
}

func (m mockedSecurityHub) DisassociateFromAdministratorAccount(_ context.Context, _ *securityhub.DisassociateFromAdministratorAccountInput, _ ...func(*securityhub.Options)) (*securityhub.DisassociateFromAdministratorAccountOutput, error) {
	return &m.DisassociateFromAdministratorAccountOutput, nil
}

func (m mockedSecurityHub) DisableSecurityHub(_ context.Context, _ *securityhub.DisableSecurityHubInput, _ ...func(*securityhub.Options)) (*securityhub.DisableSecurityHubOutput, error) {
	return &m.DisableSecurityHubOutput, nil
}

func TestSecurityHub_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	testArn := "test-arn"
	client := mockedSecurityHub{
		DescribeHubOutput: securityhub.DescribeHubOutput{
			SubscribedAt: &nowStr,
			HubArn:       aws.String(testArn),
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
			names, err := listSecurityHubs(context.Background(), client, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestSecurityHub_NukeAll(t *testing.T) {

	t.Parallel()

	client := mockedSecurityHub{
		ListMembersOutput: securityhub.ListMembersOutput{
			Members: []types.Member{{
				AccountId: aws.String("123456789012"),
			}},
		},
		DisassociateMembersOutput: securityhub.DisassociateMembersOutput{},
		DeleteMembersOutput:       securityhub.DeleteMembersOutput{},
		GetAdministratorAccountOutput: securityhub.GetAdministratorAccountOutput{
			Administrator: &types.Invitation{
				AccountId: aws.String("123456789012"),
			},
		},
		DisassociateFromAdministratorAccountOutput: securityhub.DisassociateFromAdministratorAccountOutput{},
		DisableSecurityHubOutput:                   securityhub.DisableSecurityHubOutput{},
	}

	err := deleteSecurityHubs(context.Background(), client, resource.Scope{Region: "us-east-1"}, "security-hub", []*string{aws.String("123456789012")})
	require.NoError(t, err)
}

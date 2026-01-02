package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
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

func (m mockedSecurityHub) DescribeHub(context.Context, *securityhub.DescribeHubInput, ...func(*securityhub.Options)) (*securityhub.DescribeHubOutput, error) {
	return &m.DescribeHubOutput, nil
}

func (m mockedSecurityHub) ListMembers(context.Context, *securityhub.ListMembersInput, ...func(*securityhub.Options)) (*securityhub.ListMembersOutput, error) {
	return &m.ListMembersOutput, nil
}

func (m mockedSecurityHub) DisassociateMembers(context.Context, *securityhub.DisassociateMembersInput, ...func(*securityhub.Options)) (*securityhub.DisassociateMembersOutput, error) {
	return &m.DisassociateMembersOutput, nil
}

func (m mockedSecurityHub) DeleteMembers(context.Context, *securityhub.DeleteMembersInput, ...func(*securityhub.Options)) (*securityhub.DeleteMembersOutput, error) {
	return &m.DeleteMembersOutput, nil
}

func (m mockedSecurityHub) GetAdministratorAccount(context.Context, *securityhub.GetAdministratorAccountInput, ...func(*securityhub.Options)) (*securityhub.GetAdministratorAccountOutput, error) {
	return &m.GetAdministratorAccountOutput, nil
}

func (m mockedSecurityHub) DisassociateFromAdministratorAccount(context.Context, *securityhub.DisassociateFromAdministratorAccountInput, ...func(*securityhub.Options)) (*securityhub.DisassociateFromAdministratorAccountOutput, error) {
	return &m.DisassociateFromAdministratorAccountOutput, nil
}

func (m mockedSecurityHub) DisableSecurityHub(context.Context, *securityhub.DisableSecurityHubInput, ...func(*securityhub.Options)) (*securityhub.DisableSecurityHubOutput, error) {
	return &m.DisableSecurityHubOutput, nil
}

func TestSecurityHub_List(t *testing.T) {
	t.Parallel()

	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	testArn := "arn:aws:securityhub:us-east-1:123456789012:hub/default"

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
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockedSecurityHub{
				DescribeHubOutput: securityhub.DescribeHubOutput{
					SubscribedAt: &nowStr,
					HubArn:       aws.String(testArn),
				},
			}

			result, err := listSecurityHubs(context.Background(), client, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(result))
		})
	}
}

func TestSecurityHub_Delete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		hasMembers    bool
		hasAdmin      bool
		expectedError bool
	}{
		"withMembersAndAdmin": {
			hasMembers:    true,
			hasAdmin:      true,
			expectedError: false,
		},
		"noMembersOrAdmin": {
			hasMembers:    false,
			hasAdmin:      false,
			expectedError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockedSecurityHub{
				DisableSecurityHubOutput: securityhub.DisableSecurityHubOutput{},
			}

			if tc.hasMembers {
				client.ListMembersOutput = securityhub.ListMembersOutput{
					Members: []types.Member{{AccountId: aws.String("123456789012")}},
				}
			}

			if tc.hasAdmin {
				client.GetAdministratorAccountOutput = securityhub.GetAdministratorAccountOutput{
					Administrator: &types.Invitation{AccountId: aws.String("admin-account")},
				}
			}

			ctx := context.Background()
			testArn := aws.String("arn:aws:securityhub:us-east-1:123456789012:hub/default")

			// Test each step individually
			err := removeSecurityHubMembers(ctx, client, testArn)
			require.NoError(t, err)

			err = disassociateSecurityHubAdmin(ctx, client, testArn)
			require.NoError(t, err)

			err = disableSecurityHub(ctx, client, testArn)
			require.NoError(t, err)
		})
	}
}

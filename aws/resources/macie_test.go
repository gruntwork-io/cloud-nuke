package resources

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/aws/aws-sdk-go-v2/service/macie2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockMacieClient struct {
	GetMacieSessionOutput                      macie2.GetMacieSessionOutput
	GetMacieSessionError                       error
	ListMembersOutput                          macie2.ListMembersOutput
	DisassociateMemberOutput                   macie2.DisassociateMemberOutput
	DeleteMemberOutput                         macie2.DeleteMemberOutput
	GetAdministratorAccountOutput              macie2.GetAdministratorAccountOutput
	GetAdministratorAccountError               error
	DisassociateFromAdministratorAccountOutput macie2.DisassociateFromAdministratorAccountOutput
	DisableMacieOutput                         macie2.DisableMacieOutput
	DisableMacieError                          error
}

func (m *mockMacieClient) GetMacieSession(ctx context.Context, params *macie2.GetMacieSessionInput, optFns ...func(*macie2.Options)) (*macie2.GetMacieSessionOutput, error) {
	return &m.GetMacieSessionOutput, m.GetMacieSessionError
}

func (m *mockMacieClient) ListMembers(ctx context.Context, params *macie2.ListMembersInput, optFns ...func(*macie2.Options)) (*macie2.ListMembersOutput, error) {
	return &m.ListMembersOutput, nil
}

func (m *mockMacieClient) DisassociateMember(ctx context.Context, params *macie2.DisassociateMemberInput, optFns ...func(*macie2.Options)) (*macie2.DisassociateMemberOutput, error) {
	return &m.DisassociateMemberOutput, nil
}

func (m *mockMacieClient) DeleteMember(ctx context.Context, params *macie2.DeleteMemberInput, optFns ...func(*macie2.Options)) (*macie2.DeleteMemberOutput, error) {
	return &m.DeleteMemberOutput, nil
}

func (m *mockMacieClient) GetAdministratorAccount(ctx context.Context, params *macie2.GetAdministratorAccountInput, optFns ...func(*macie2.Options)) (*macie2.GetAdministratorAccountOutput, error) {
	return &m.GetAdministratorAccountOutput, m.GetAdministratorAccountError
}

func (m *mockMacieClient) DisassociateFromAdministratorAccount(ctx context.Context, params *macie2.DisassociateFromAdministratorAccountInput, optFns ...func(*macie2.Options)) (*macie2.DisassociateFromAdministratorAccountOutput, error) {
	return &m.DisassociateFromAdministratorAccountOutput, nil
}

func (m *mockMacieClient) DisableMacie(ctx context.Context, params *macie2.DisableMacieInput, optFns ...func(*macie2.Options)) (*macie2.DisableMacieOutput, error) {
	return &m.DisableMacieOutput, m.DisableMacieError
}

func TestListMacieSessions(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testStatus := "ENABLED"

	tests := map[string]struct {
		mock      *mockMacieClient
		configObj config.ResourceType
		expected  []string
	}{
		"enabled session - no filter": {
			mock: &mockMacieClient{
				GetMacieSessionOutput: macie2.GetMacieSessionOutput{
					Status:    types.MacieStatusEnabled,
					CreatedAt: aws.Time(now),
				},
			},
			configObj: config.ResourceType{},
			expected:  []string{testStatus},
		},
		"session excluded by time filter": {
			mock: &mockMacieClient{
				GetMacieSessionOutput: macie2.GetMacieSessionOutput{
					Status:    types.MacieStatusEnabled,
					CreatedAt: aws.Time(now.Add(1 * time.Hour)),
				},
			},
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{},
		},
		"macie not enabled - returns nil": {
			mock: &mockMacieClient{
				GetMacieSessionError: errors.New("Macie is not enabled"),
			},
			configObj: config.ResourceType{},
			expected:  []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results, err := listMacieSessions(context.Background(), tc.mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(results))
		})
	}
}

func TestDeleteMacieSessions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mock        *mockMacieClient
		identifiers []*string
		expectError bool
	}{
		"successful deletion with members and admin": {
			mock: &mockMacieClient{
				ListMembersOutput: macie2.ListMembersOutput{
					Members: []types.Member{
						{AccountId: aws.String("123456789012")},
						{AccountId: aws.String("987654321098")},
					},
				},
				GetAdministratorAccountOutput: macie2.GetAdministratorAccountOutput{
					Administrator: &types.Invitation{
						AccountId: aws.String("111111111111"),
					},
				},
			},
			identifiers: []*string{aws.String("ENABLED")},
			expectError: false,
		},
		"successful deletion without members or admin": {
			mock: &mockMacieClient{
				ListMembersOutput:             macie2.ListMembersOutput{},
				GetAdministratorAccountOutput: macie2.GetAdministratorAccountOutput{},
			},
			identifiers: []*string{aws.String("ENABLED")},
			expectError: false,
		},
		"empty identifiers returns nil": {
			mock:        &mockMacieClient{},
			identifiers: []*string{},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results := deleteMacieSessions(context.Background(), tc.mock, resource.Scope{Region: "us-east-1"}, "macie-member", tc.identifiers)

			if len(tc.identifiers) == 0 {
				require.Nil(t, results)
			} else {
				require.Len(t, results, 1)
				if tc.expectError {
					require.Error(t, results[0].Error)
				} else {
					require.NoError(t, results[0].Error)
				}
			}
		})
	}
}

func TestListMacieSessions_TimeInclusionFilter(t *testing.T) {
	t.Parallel()

	// Macie doesn't have names to filter by, so test time-based inclusion works
	now := time.Now()

	mock := &mockMacieClient{
		GetMacieSessionOutput: macie2.GetMacieSessionOutput{
			Status:    types.MacieStatusEnabled,
			CreatedAt: aws.Time(now),
		},
	}

	// Test that time inclusion works - include sessions created before now+1h
	results, err := listMacieSessions(context.Background(), mock, resource.Scope{}, config.ResourceType{
		IncludeRule: config.FilterRule{
			TimeBefore: aws.Time(now.Add(1 * time.Hour)),
		},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "ENABLED", aws.ToString(results[0]))
}

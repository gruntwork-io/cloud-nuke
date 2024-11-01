package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedIAMRoles struct {
	IAMRolesAPI
	ListAttachedRolePoliciesOutput      iam.ListAttachedRolePoliciesOutput
	ListInstanceProfilesForRoleOutput   iam.ListInstanceProfilesForRoleOutput
	ListRolePoliciesOutput              iam.ListRolePoliciesOutput
	ListRolesOutput                     iam.ListRolesOutput
	DeleteInstanceProfileOutput         iam.DeleteInstanceProfileOutput
	DetachRolePolicyOutput              iam.DetachRolePolicyOutput
	DeleteRolePolicyOutput              iam.DeleteRolePolicyOutput
	DeleteRoleOutput                    iam.DeleteRoleOutput
	RemoveRoleFromInstanceProfileOutput iam.RemoveRoleFromInstanceProfileOutput
}

func (m mockedIAMRoles) ListAttachedRolePolicies(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error) {
	return &m.ListAttachedRolePoliciesOutput, nil
}

func (m mockedIAMRoles) ListInstanceProfilesForRole(ctx context.Context, params *iam.ListInstanceProfilesForRoleInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfilesForRoleOutput, error) {
	return &m.ListInstanceProfilesForRoleOutput, nil
}

func (m mockedIAMRoles) ListRolePolicies(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error) {
	return &m.ListRolePoliciesOutput, nil
}

func (m mockedIAMRoles) ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error) {
	return &m.ListRolesOutput, nil
}

func (m mockedIAMRoles) DeleteInstanceProfile(ctx context.Context, params *iam.DeleteInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.DeleteInstanceProfileOutput, error) {
	return &m.DeleteInstanceProfileOutput, nil
}

func (m mockedIAMRoles) DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
	return &m.DetachRolePolicyOutput, nil
}

func (m mockedIAMRoles) DeleteRolePolicy(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error) {
	return &m.DeleteRolePolicyOutput, nil
}

func (m mockedIAMRoles) DeleteRole(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
	return &m.DeleteRoleOutput, nil
}

func (m mockedIAMRoles) RemoveRoleFromInstanceProfile(ctx context.Context, params *iam.RemoveRoleFromInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.RemoveRoleFromInstanceProfileOutput, error) {
	return &m.RemoveRoleFromInstanceProfileOutput, nil
}

func TestIAMRoles_GetAll(t *testing.T) {
	t.Parallel()
	testName1 := "test-role1"
	testName2 := "test-role2"
	now := time.Now()
	ir := IAMRoles{
		Client: mockedIAMRoles{
			ListRolesOutput: iam.ListRolesOutput{
				Roles: []types.Role{
					{
						RoleName:   aws.String(testName1),
						CreateDate: aws.Time(now),
					},
					{
						RoleName:   aws.String(testName2),
						CreateDate: aws.Time(now.Add(1)),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ir.getAll(context.Background(), config.Config{
				IAMRoles: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestIAMRoles_NukeAll(t *testing.T) {
	t.Parallel()
	ir := IAMRoles{
		Client: mockedIAMRoles{
			ListInstanceProfilesForRoleOutput: iam.ListInstanceProfilesForRoleOutput{
				InstanceProfiles: []types.InstanceProfile{
					{
						InstanceProfileName: aws.String("test-instance-profile"),
					},
				},
			},
			RemoveRoleFromInstanceProfileOutput: iam.RemoveRoleFromInstanceProfileOutput{},
			DeleteInstanceProfileOutput:         iam.DeleteInstanceProfileOutput{},
			ListRolePoliciesOutput: iam.ListRolePoliciesOutput{
				PolicyNames: []string{
					"test-policy",
				},
			},
			DeleteRolePolicyOutput: iam.DeleteRolePolicyOutput{},
			ListAttachedRolePoliciesOutput: iam.ListAttachedRolePoliciesOutput{
				AttachedPolicies: []types.AttachedPolicy{
					{
						PolicyArn: aws.String("test-policy-arn"),
					},
				},
			},
			DetachRolePolicyOutput: iam.DetachRolePolicyOutput{},
			DeleteRoleOutput:       iam.DeleteRoleOutput{},
		},
	}

	err := ir.nukeAll([]*string{aws.String("test-role")})
	require.NoError(t, err)
}

package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedIAMRoles struct {
	iamiface.IAMAPI
	ListRolesPagesOutput                iam.ListRolesOutput
	ListInstanceProfilesForRoleOutput   iam.ListInstanceProfilesForRoleOutput
	RemoveRoleFromInstanceProfileOutput iam.RemoveRoleFromInstanceProfileOutput
	DeleteInstanceProfileOutput         iam.DeleteInstanceProfileOutput
	ListRolePoliciesOutput              iam.ListRolePoliciesOutput
	DeleteRolePolicyOutput              iam.DeleteRolePolicyOutput
	ListAttachedRolePoliciesOutput      iam.ListAttachedRolePoliciesOutput
	DetachRolePolicyOutput              iam.DetachRolePolicyOutput
	DeleteRoleOutput                    iam.DeleteRoleOutput
}

func (m mockedIAMRoles) ListRolesPagesWithContext(_ aws.Context, input *iam.ListRolesInput, f func(*iam.ListRolesOutput, bool) bool, _ ...request.Option) error {
	f(&m.ListRolesPagesOutput, true)
	return nil
}

func (m mockedIAMRoles) ListInstanceProfilesForRoleWithContext(_ aws.Context, input *iam.ListInstanceProfilesForRoleInput, _ ...request.Option) (*iam.ListInstanceProfilesForRoleOutput, error) {
	return &m.ListInstanceProfilesForRoleOutput, nil
}

func (m mockedIAMRoles) RemoveRoleFromInstanceProfileWithContext(_ aws.Context, input *iam.RemoveRoleFromInstanceProfileInput, _ ...request.Option) (*iam.RemoveRoleFromInstanceProfileOutput, error) {
	return &m.RemoveRoleFromInstanceProfileOutput, nil
}

func (m mockedIAMRoles) DeleteInstanceProfileWithContext(_ aws.Context, input *iam.DeleteInstanceProfileInput, _ ...request.Option) (*iam.DeleteInstanceProfileOutput, error) {
	return &m.DeleteInstanceProfileOutput, nil
}

func (m mockedIAMRoles) ListRolePoliciesWithContext(_ aws.Context, input *iam.ListRolePoliciesInput, _ ...request.Option) (*iam.ListRolePoliciesOutput, error) {
	return &m.ListRolePoliciesOutput, nil
}

func (m mockedIAMRoles) DeleteRolePolicyWithContext(_ aws.Context, input *iam.DeleteRolePolicyInput, _ ...request.Option) (*iam.DeleteRolePolicyOutput, error) {
	return &m.DeleteRolePolicyOutput, nil
}

func (m mockedIAMRoles) ListAttachedRolePoliciesWithContext(_ aws.Context, input *iam.ListAttachedRolePoliciesInput, _ ...request.Option) (*iam.ListAttachedRolePoliciesOutput, error) {
	return &m.ListAttachedRolePoliciesOutput, nil
}

func (m mockedIAMRoles) DetachRolePolicyWithContext(_ aws.Context, input *iam.DetachRolePolicyInput, _ ...request.Option) (*iam.DetachRolePolicyOutput, error) {
	return &m.DetachRolePolicyOutput, nil
}

func (m mockedIAMRoles) DeleteRoleWithContext(_ aws.Context, input *iam.DeleteRoleInput, _ ...request.Option) (*iam.DeleteRoleOutput, error) {
	return &m.DeleteRoleOutput, nil
}

func TestIAMRoles_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-role1"
	testName2 := "test-role2"
	now := time.Now()
	ir := IAMRoles{
		Client: mockedIAMRoles{
			ListRolesPagesOutput: iam.ListRolesOutput{
				Roles: []*iam.Role{
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestIAMRoles_NukeAll(t *testing.T) {

	t.Parallel()

	ir := IAMRoles{
		Client: mockedIAMRoles{
			ListInstanceProfilesForRoleOutput: iam.ListInstanceProfilesForRoleOutput{
				InstanceProfiles: []*iam.InstanceProfile{
					{
						InstanceProfileName: aws.String("test-instance-profile"),
					},
				},
			},
			RemoveRoleFromInstanceProfileOutput: iam.RemoveRoleFromInstanceProfileOutput{},
			DeleteInstanceProfileOutput:         iam.DeleteInstanceProfileOutput{},
			ListRolePoliciesOutput: iam.ListRolePoliciesOutput{
				PolicyNames: []*string{
					aws.String("test-policy"),
				},
			},
			DeleteRolePolicyOutput: iam.DeleteRolePolicyOutput{},
			ListAttachedRolePoliciesOutput: iam.ListAttachedRolePoliciesOutput{
				AttachedPolicies: []*iam.AttachedPolicy{
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

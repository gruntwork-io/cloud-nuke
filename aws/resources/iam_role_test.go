package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
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

func (m mockedIAMRoles) ListRolesPages(input *iam.ListRolesInput, f func(*iam.ListRolesOutput, bool) bool) error {
	f(&m.ListRolesPagesOutput, true)
	return nil
}

func (m mockedIAMRoles) ListInstanceProfilesForRole(input *iam.ListInstanceProfilesForRoleInput) (*iam.ListInstanceProfilesForRoleOutput, error) {
	return &m.ListInstanceProfilesForRoleOutput, nil
}

func (m mockedIAMRoles) RemoveRoleFromInstanceProfile(input *iam.RemoveRoleFromInstanceProfileInput) (*iam.RemoveRoleFromInstanceProfileOutput, error) {
	return &m.RemoveRoleFromInstanceProfileOutput, nil
}

func (m mockedIAMRoles) DeleteInstanceProfile(input *iam.DeleteInstanceProfileInput) (*iam.DeleteInstanceProfileOutput, error) {
	return &m.DeleteInstanceProfileOutput, nil
}

func (m mockedIAMRoles) ListRolePolicies(input *iam.ListRolePoliciesInput) (*iam.ListRolePoliciesOutput, error) {
	return &m.ListRolePoliciesOutput, nil
}

func (m mockedIAMRoles) DeleteRolePolicy(input *iam.DeleteRolePolicyInput) (*iam.DeleteRolePolicyOutput, error) {
	return &m.DeleteRolePolicyOutput, nil
}

func (m mockedIAMRoles) ListAttachedRolePolicies(input *iam.ListAttachedRolePoliciesInput) (*iam.ListAttachedRolePoliciesOutput, error) {
	return &m.ListAttachedRolePoliciesOutput, nil
}

func (m mockedIAMRoles) DetachRolePolicy(input *iam.DetachRolePolicyInput) (*iam.DetachRolePolicyOutput, error) {
	return &m.DetachRolePolicyOutput, nil
}

func (m mockedIAMRoles) DeleteRole(input *iam.DeleteRoleInput) (*iam.DeleteRoleOutput, error) {
	return &m.DeleteRoleOutput, nil
}

func TestIAMRoles_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
			names, err := ir.getAll(config.Config{
				IAMRoles: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestIAMRoles_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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

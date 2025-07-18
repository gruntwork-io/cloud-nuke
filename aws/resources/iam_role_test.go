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
	ListRoleTagsOutputByName            map[string]*iam.ListRoleTagsOutput
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

func (m mockedIAMRoles) ListRoleTags(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error) {
	return m.ListRoleTagsOutputByName[*params.RoleName], nil
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
			ListRoleTagsOutputByName: map[string]*iam.ListRoleTagsOutput{
				testName1: {Tags: []types.Tag{{Key: aws.String("foo"), Value: aws.String("bar")}}},
				testName2: {Tags: []types.Tag{{Key: aws.String("faz"), Value: aws.String("baz")}}},
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
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
				}},
			expected: []string{testName2},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
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

func TestIAMRoles_ServiceLinkedRoles(t *testing.T) {
	t.Parallel()
	now := time.Now()
	ir := IAMRoles{}
	configObj := config.Config{IAMRoles: config.ResourceType{}}

	tests := []struct {
		name     string
		role     types.Role
		expected bool
	}{
		{
			name: "regular role should be included",
			role: types.Role{
				RoleName:   aws.String("MyCustomRole"),
				Arn:        aws.String("arn:aws:iam::123456789012:role/MyCustomRole"),
				CreateDate: aws.Time(now),
			},
			expected: true,
		},
		{
			name: "AWSServiceRoleForTrustedAdvisor should be filtered out",
			role: types.Role{
				RoleName:   aws.String("AWSServiceRoleForTrustedAdvisor"),
				Arn:        aws.String("arn:aws:iam::123456789012:role/aws-service-role/trustedadvisor.amazonaws.com/AWSServiceRoleForTrustedAdvisor"),
				CreateDate: aws.Time(now),
			},
			expected: false,
		},
		{
			name: "AWSServiceRoleForSupport should be filtered out",
			role: types.Role{
				RoleName:   aws.String("AWSServiceRoleForSupport"),
				Arn:        aws.String("arn:aws:iam::123456789012:role/aws-service-role/support.amazonaws.com/AWSServiceRoleForSupport"),
				CreateDate: aws.Time(now),
			},
			expected: false,
		},
		{
			name: "AWSServiceRoleForAmazonCodeGuruReviewer should be filtered out",
			role: types.Role{
				RoleName:   aws.String("AWSServiceRoleForAmazonCodeGuruReviewer"),
				Arn:        aws.String("arn:aws:iam::123456789012:role/aws-service-role/codeguru-reviewer.amazonaws.com/AWSServiceRoleForAmazonCodeGuruReviewer"),
				CreateDate: aws.Time(now),
			},
			expected: false,
		},
		{
			name: "OrganizationAccountAccessRole should be filtered out",
			role: types.Role{
				RoleName:   aws.String("OrganizationAccountAccessRole"),
				Arn:        aws.String("arn:aws:iam::123456789012:role/OrganizationAccountAccessRole"),
				CreateDate: aws.Time(now),
			},
			expected: false,
		},
		{
			name: "aws-reserved role should be filtered out",
			role: types.Role{
				RoleName:   aws.String("AWSReservedSSO_TestRole"),
				Arn:        aws.String("arn:aws:iam::123456789012:role/aws-reserved/sso.amazonaws.com/AWSReservedSSO_TestRole"),
				CreateDate: aws.Time(now),
			},
			expected: false,
		},
		{
			name:     "nil role should be filtered out",
			role:     types.Role{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rolePtr *types.Role
			if tt.role.RoleName != nil || tt.role.Arn != nil {
				rolePtr = &tt.role
			}
			result := ir.shouldInclude(rolePtr, configObj, []types.Tag{})
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIAMRoles_GetAll_ServiceLinkedRolesFiltered(t *testing.T) {
	t.Parallel()
	now := time.Now()
	ir := IAMRoles{
		Client: mockedIAMRoles{
			ListRolesOutput: iam.ListRolesOutput{
				Roles: []types.Role{
					{
						RoleName:   aws.String("MyCustomRole"),
						Arn:        aws.String("arn:aws:iam::123456789012:role/MyCustomRole"),
						CreateDate: aws.Time(now),
					},
					{
						RoleName:   aws.String("AWSServiceRoleForTrustedAdvisor"),
						Arn:        aws.String("arn:aws:iam::123456789012:role/aws-service-role/trustedadvisor.amazonaws.com/AWSServiceRoleForTrustedAdvisor"),
						CreateDate: aws.Time(now),
					},
					{
						RoleName:   aws.String("AWSServiceRoleForSupport"),
						Arn:        aws.String("arn:aws:iam::123456789012:role/aws-service-role/support.amazonaws.com/AWSServiceRoleForSupport"),
						CreateDate: aws.Time(now),
					},
					{
						RoleName:   aws.String("AnotherCustomRole"),
						Arn:        aws.String("arn:aws:iam::123456789012:role/AnotherCustomRole"),
						CreateDate: aws.Time(now),
					},
				},
			},
		},
	}

	roles, err := ir.getAll(context.Background(), config.Config{
		IAMRoles: config.ResourceType{},
	})

	require.NoError(t, err)
	// Should only return custom roles, not service-linked roles
	expected := []string{"MyCustomRole", "AnotherCustomRole"}
	require.Equal(t, expected, aws.ToStringSlice(roles))
}

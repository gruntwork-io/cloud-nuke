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

type mockedIAMPolicies struct {
	IAMPoliciesAPI
	ListEntitiesForPolicyOutput             iam.ListEntitiesForPolicyOutput
	ListEntitiesForPolicyPermBoundaryOutput iam.ListEntitiesForPolicyOutput
	ListPoliciesOutput                      iam.ListPoliciesOutput
	ListPolicyTagsOutputByArn               map[string]*iam.ListPolicyTagsOutput
	ListPolicyVersionsOutput                iam.ListPolicyVersionsOutput
	DeletePolicyOutput                      iam.DeletePolicyOutput
	DeletePolicyVersionOutput               iam.DeletePolicyVersionOutput
	DetachGroupPolicyOutput                 iam.DetachGroupPolicyOutput
	DetachUserPolicyOutput                  iam.DetachUserPolicyOutput
	DetachRolePolicyOutput                  iam.DetachRolePolicyOutput
	DeleteUserPermissionsBoundaryOutput     iam.DeleteUserPermissionsBoundaryOutput
	DeleteRolePermissionsBoundaryOutput     iam.DeleteRolePermissionsBoundaryOutput
}

var _ IAMPoliciesAPI = (*mockedIAMPolicies)(nil)

func (m mockedIAMPolicies) ListEntitiesForPolicy(ctx context.Context, params *iam.ListEntitiesForPolicyInput, optFns ...func(*iam.Options)) (*iam.ListEntitiesForPolicyOutput, error) {
	// Return different outputs based on PolicyUsageFilter
	if params.PolicyUsageFilter == types.PolicyUsageTypePermissionsBoundary {
		return &m.ListEntitiesForPolicyPermBoundaryOutput, nil
	}
	return &m.ListEntitiesForPolicyOutput, nil
}

func (m mockedIAMPolicies) ListPolicies(ctx context.Context, params *iam.ListPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListPoliciesOutput, error) {
	return &m.ListPoliciesOutput, nil
}

func (m mockedIAMPolicies) ListPolicyTags(ctx context.Context, params *iam.ListPolicyTagsInput, optFns ...func(*iam.Options)) (*iam.ListPolicyTagsOutput, error) {
	return m.ListPolicyTagsOutputByArn[*params.PolicyArn], nil
}

func (m mockedIAMPolicies) ListPolicyVersions(ctx context.Context, params *iam.ListPolicyVersionsInput, optFns ...func(*iam.Options)) (*iam.ListPolicyVersionsOutput, error) {
	return &m.ListPolicyVersionsOutput, nil
}

func (m mockedIAMPolicies) DeletePolicy(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error) {
	return &m.DeletePolicyOutput, nil
}

func (m mockedIAMPolicies) DeletePolicyVersion(ctx context.Context, params *iam.DeletePolicyVersionInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyVersionOutput, error) {
	return &m.DeletePolicyVersionOutput, nil
}

func (m mockedIAMPolicies) DetachGroupPolicy(ctx context.Context, params *iam.DetachGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachGroupPolicyOutput, error) {
	return &m.DetachGroupPolicyOutput, nil
}

func (m mockedIAMPolicies) DetachUserPolicy(ctx context.Context, params *iam.DetachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachUserPolicyOutput, error) {
	return &m.DetachUserPolicyOutput, nil
}

func (m mockedIAMPolicies) DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
	return &m.DetachRolePolicyOutput, nil
}

func (m mockedIAMPolicies) DeleteUserPermissionsBoundary(ctx context.Context, params *iam.DeleteUserPermissionsBoundaryInput, optFns ...func(*iam.Options)) (*iam.DeleteUserPermissionsBoundaryOutput, error) {
	return &m.DeleteUserPermissionsBoundaryOutput, nil
}

func (m mockedIAMPolicies) DeleteRolePermissionsBoundary(ctx context.Context, params *iam.DeleteRolePermissionsBoundaryInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePermissionsBoundaryOutput, error) {
	return &m.DeleteRolePermissionsBoundaryOutput, nil
}

func TestIAMPolicy_GetAll(t *testing.T) {
	t.Parallel()
	testName1 := "MyPolicy1"
	testName2 := "MyPolicy2"
	testArn1 := "arn:aws:iam::123456789012:policy/MyPolicy1"
	testArn2 := "arn:aws:iam::123456789012:policy/MyPolicy2"
	now := time.Now()
	ip := IAMPolicies{
		Client: mockedIAMPolicies{
			ListPoliciesOutput: iam.ListPoliciesOutput{
				Policies: []types.Policy{
					{
						Arn:        aws.String(testArn1),
						PolicyName: aws.String(testName1),
						CreateDate: aws.Time(now),
					},
					{
						Arn:        aws.String(testArn2),
						PolicyName: aws.String(testName2),
						CreateDate: aws.Time(now.Add(1)),
					},
				},
			},
			ListPolicyTagsOutputByArn: map[string]*iam.ListPolicyTagsOutput{
				testArn1: {Tags: []types.Tag{{Key: aws.String("foo"), Value: aws.String("bar")}}},
				testArn2: {Tags: []types.Tag{{Key: aws.String("faz"), Value: aws.String("baz")}}},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testArn1},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
				},
			},
			expected: []string{testArn2},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
				},
			},
			expected: []string{testArn1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ip.getAll(context.Background(), config.Config{
				IAMPolicies: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestIAMPolicy_NukeAll(t *testing.T) {
	t.Parallel()

	ip := IAMPolicies{
		Client: mockedIAMPolicies{
			ListEntitiesForPolicyOutput: iam.ListEntitiesForPolicyOutput{
				PolicyGroups: []types.PolicyGroup{
					{GroupName: aws.String("group1")},
				},
				PolicyUsers: []types.PolicyUser{
					{UserName: aws.String("user1")},
				},
				PolicyRoles: []types.PolicyRole{
					{RoleName: aws.String("role1")},
				},
			},
			ListEntitiesForPolicyPermBoundaryOutput: iam.ListEntitiesForPolicyOutput{},
			DetachUserPolicyOutput:                  iam.DetachUserPolicyOutput{},
			DetachGroupPolicyOutput:                 iam.DetachGroupPolicyOutput{},
			DetachRolePolicyOutput:                  iam.DetachRolePolicyOutput{},
			ListPolicyVersionsOutput: iam.ListPolicyVersionsOutput{
				Versions: []types.PolicyVersion{
					{
						VersionId:        aws.String("v1"),
						IsDefaultVersion: false,
					},
				},
			},
			DeletePolicyVersionOutput: iam.DeletePolicyVersionOutput{},
			DeletePolicyOutput:        iam.DeletePolicyOutput{},
		},
	}

	err := ip.nukeAll([]*string{aws.String("arn:aws:iam::123456789012:policy/MyPolicy1")})
	require.NoError(t, err)
}

func TestIAMPolicy_NukeAll_WithPermissionsBoundary(t *testing.T) {
	t.Parallel()

	ip := IAMPolicies{
		Client: mockedIAMPolicies{
			// Regular policy attachments
			ListEntitiesForPolicyOutput: iam.ListEntitiesForPolicyOutput{
				PolicyRoles: []types.PolicyRole{
					{RoleName: aws.String("role-with-policy")},
				},
			},
			// Permissions boundary attachments
			ListEntitiesForPolicyPermBoundaryOutput: iam.ListEntitiesForPolicyOutput{
				PolicyUsers: []types.PolicyUser{
					{UserName: aws.String("user-with-boundary")},
				},
				PolicyRoles: []types.PolicyRole{
					{RoleName: aws.String("role-with-boundary")},
				},
			},
			DetachRolePolicyOutput:              iam.DetachRolePolicyOutput{},
			DeleteUserPermissionsBoundaryOutput: iam.DeleteUserPermissionsBoundaryOutput{},
			DeleteRolePermissionsBoundaryOutput: iam.DeleteRolePermissionsBoundaryOutput{},
			ListPolicyVersionsOutput: iam.ListPolicyVersionsOutput{
				Versions: []types.PolicyVersion{
					{
						VersionId:        aws.String("v1"),
						IsDefaultVersion: true,
					},
				},
			},
			DeletePolicyOutput: iam.DeletePolicyOutput{},
		},
	}

	err := ip.nukeAll([]*string{aws.String("arn:aws:iam::123456789012:policy/BoundaryPolicy")})
	require.NoError(t, err)
}

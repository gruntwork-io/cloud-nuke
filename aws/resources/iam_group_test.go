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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedIAMGroups struct {
	IAMGroupsAPI
	DetachGroupPolicyOutput         iam.DetachGroupPolicyOutput
	DeleteGroupPolicyOutput         iam.DeleteGroupPolicyOutput
	DeleteGroupOutput               iam.DeleteGroupOutput
	GetGroupOutput                  iam.GetGroupOutput
	ListAttachedGroupPoliciesOutput iam.ListAttachedGroupPoliciesOutput
	ListGroupsOutput                iam.ListGroupsOutput
	ListGroupPoliciesOutput         iam.ListGroupPoliciesOutput
	RemoveUserFromGroupOutput       iam.RemoveUserFromGroupOutput
}

func (m mockedIAMGroups) DetachGroupPolicy(ctx context.Context, params *iam.DetachGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachGroupPolicyOutput, error) {
	return &m.DetachGroupPolicyOutput, nil
}

func (m mockedIAMGroups) DeleteGroupPolicy(ctx context.Context, params *iam.DeleteGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteGroupPolicyOutput, error) {
	return &m.DeleteGroupPolicyOutput, nil
}

func (m mockedIAMGroups) DeleteGroup(ctx context.Context, params *iam.DeleteGroupInput, optFns ...func(*iam.Options)) (*iam.DeleteGroupOutput, error) {
	return &m.DeleteGroupOutput, nil
}

func (m mockedIAMGroups) GetGroup(ctx context.Context, params *iam.GetGroupInput, optFns ...func(*iam.Options)) (*iam.GetGroupOutput, error) {
	return &m.GetGroupOutput, nil
}

func (m mockedIAMGroups) ListAttachedGroupPolicies(ctx context.Context, params *iam.ListAttachedGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedGroupPoliciesOutput, error) {
	return &m.ListAttachedGroupPoliciesOutput, nil
}

func (m mockedIAMGroups) ListGroups(ctx context.Context, params *iam.ListGroupsInput, optFns ...func(*iam.Options)) (*iam.ListGroupsOutput, error) {
	return &m.ListGroupsOutput, nil
}

func (m mockedIAMGroups) ListGroupPolicies(ctx context.Context, params *iam.ListGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListGroupPoliciesOutput, error) {
	return &m.ListGroupPoliciesOutput, nil
}

func (m mockedIAMGroups) RemoveUserFromGroup(ctx context.Context, params *iam.RemoveUserFromGroupInput, optFns ...func(*iam.Options)) (*iam.RemoveUserFromGroupOutput, error) {
	return &m.RemoveUserFromGroupOutput, nil
}

func TestIamGroups_GetAll(t *testing.T) {
	t.Parallel()
	testName1 := "group1"
	testName2 := "group2"
	now := time.Now()
	client := mockedIAMGroups{
		ListGroupsOutput: iam.ListGroupsOutput{
			Groups: []types.Group{
				{
					GroupName:  aws.String(testName1),
					CreateDate: aws.Time(now),
				},
				{
					GroupName:  aws.String(testName2),
					CreateDate: aws.Time(now.Add(1)),
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
			names, err := listIAMGroups(context.Background(), client, resource.Scope{Region: "global"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestIamGroups_DeleteIAMGroup(t *testing.T) {
	t.Parallel()
	client := mockedIAMGroups{
		GetGroupOutput: iam.GetGroupOutput{
			Users: []types.User{
				{
					UserName: aws.String("user1"),
				},
			},
		},
		RemoveUserFromGroupOutput: iam.RemoveUserFromGroupOutput{},
		ListAttachedGroupPoliciesOutput: iam.ListAttachedGroupPoliciesOutput{
			AttachedPolicies: []types.AttachedPolicy{
				{
					PolicyName: aws.String("policy1"),
					PolicyArn:  aws.String("arn:aws:iam::123456789012:policy/policy1"),
				},
			},
		},
		DetachGroupPolicyOutput: iam.DetachGroupPolicyOutput{},
		ListGroupPoliciesOutput: iam.ListGroupPoliciesOutput{
			PolicyNames: []string{
				"inline-policy1",
			},
		},
		DeleteGroupPolicyOutput: iam.DeleteGroupPolicyOutput{},
		DeleteGroupOutput:       iam.DeleteGroupOutput{},
	}

	err := deleteIAMGroup(context.Background(), client, aws.String("group1"))
	require.NoError(t, err)
}

func TestIamGroups_RemoveUsersFromGroup(t *testing.T) {
	t.Parallel()
	client := mockedIAMGroups{
		GetGroupOutput: iam.GetGroupOutput{
			Users: []types.User{
				{UserName: aws.String("user1")},
				{UserName: aws.String("user2")},
			},
		},
		RemoveUserFromGroupOutput: iam.RemoveUserFromGroupOutput{},
	}

	err := removeUsersFromGroup(context.Background(), client, aws.String("test-group"))
	require.NoError(t, err)
}

func TestIamGroups_DetachGroupPolicies(t *testing.T) {
	t.Parallel()
	client := mockedIAMGroups{
		ListAttachedGroupPoliciesOutput: iam.ListAttachedGroupPoliciesOutput{
			AttachedPolicies: []types.AttachedPolicy{
				{
					PolicyName: aws.String("policy1"),
					PolicyArn:  aws.String("arn:aws:iam::123456789012:policy/policy1"),
				},
			},
		},
		DetachGroupPolicyOutput: iam.DetachGroupPolicyOutput{},
	}

	err := detachGroupPolicies(context.Background(), client, aws.String("test-group"))
	require.NoError(t, err)
}

func TestIamGroups_DeleteGroupInlinePolicies(t *testing.T) {
	t.Parallel()
	client := mockedIAMGroups{
		ListGroupPoliciesOutput: iam.ListGroupPoliciesOutput{
			PolicyNames: []string{"inline-policy1", "inline-policy2"},
		},
		DeleteGroupPolicyOutput: iam.DeleteGroupPolicyOutput{},
	}

	err := deleteGroupInlinePolicies(context.Background(), client, aws.String("test-group"))
	require.NoError(t, err)
}

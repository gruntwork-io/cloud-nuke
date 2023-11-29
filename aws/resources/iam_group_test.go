package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

type mockedIAMGroups struct {
	iamiface.IAMAPI
	ListGroupsPagesOutput                iam.ListGroupsOutput
	GetGroupOutput                       iam.GetGroupOutput
	RemoveUserFromGroupOutput            iam.RemoveUserFromGroupOutput
	DeleteGroupOutput                    iam.DeleteGroupOutput
	ListAttachedGroupPoliciesPagesOutput iam.ListAttachedGroupPoliciesOutput
	DetachGroupPolicyOutput              iam.DetachGroupPolicyOutput
	ListGroupPoliciesOutput              iam.ListGroupPoliciesOutput
	DeleteGroupPolicyOutput              iam.DeleteGroupPolicyOutput
}

func (m mockedIAMGroups) ListGroupsPages(input *iam.ListGroupsInput, fn func(*iam.ListGroupsOutput, bool) bool) error {
	fn(&m.ListGroupsPagesOutput, true)
	return nil
}

func (m mockedIAMGroups) DeleteGroup(input *iam.DeleteGroupInput) (*iam.DeleteGroupOutput, error) {
	return &m.DeleteGroupOutput, nil
}

func (m mockedIAMGroups) ListAttachedGroupPoliciesPages(input *iam.ListAttachedGroupPoliciesInput, fn func(*iam.ListAttachedGroupPoliciesOutput, bool) bool) error {
	fn(&m.ListAttachedGroupPoliciesPagesOutput, true)
	return nil
}

func (m mockedIAMGroups) ListGroupPoliciesPages(input *iam.ListGroupPoliciesInput, fn func(*iam.ListGroupPoliciesOutput, bool) bool) error {
	fn(&m.ListGroupPoliciesOutput, true)
	return nil
}

func (m mockedIAMGroups) DetachGroupPolicy(input *iam.DetachGroupPolicyInput) (*iam.DetachGroupPolicyOutput, error) {
	return &m.DetachGroupPolicyOutput, nil
}

func (m mockedIAMGroups) DeleteGroupPolicy(input *iam.DeleteGroupPolicyInput) (*iam.DeleteGroupPolicyOutput, error) {
	return &m.DeleteGroupPolicyOutput, nil
}

func (m mockedIAMGroups) GetGroup(input *iam.GetGroupInput) (*iam.GetGroupOutput, error) {
	return &m.GetGroupOutput, nil
}

func (m mockedIAMGroups) RemoveUserFromGroup(input *iam.RemoveUserFromGroupInput) (*iam.RemoveUserFromGroupOutput, error) {
	return &m.RemoveUserFromGroupOutput, nil
}

func TestIamGroups_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "group1"
	testName2 := "group2"
	now := time.Now()
	ig := IAMGroups{
		Client: mockedIAMGroups{
			ListGroupsPagesOutput: iam.ListGroupsOutput{
				Groups: []*iam.Group{
					{
						GroupName:  awsgo.String(testName1),
						CreateDate: awsgo.Time(now),
					},
					{
						GroupName:  awsgo.String(testName2),
						CreateDate: awsgo.Time(now.Add(1)),
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
					TimeAfter: awsgo.Time(now),
				}},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ig.getAll(context.Background(), config.Config{
				IAMGroups: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestIamGroups_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	ig := IAMGroups{
		Client: mockedIAMGroups{
			GetGroupOutput: iam.GetGroupOutput{
				Users: []*iam.User{
					{
						UserName: awsgo.String("user1"),
					},
				},
			},
			RemoveUserFromGroupOutput: iam.RemoveUserFromGroupOutput{},
			ListAttachedGroupPoliciesPagesOutput: iam.ListAttachedGroupPoliciesOutput{
				AttachedPolicies: []*iam.AttachedPolicy{
					{
						PolicyName: awsgo.String("policy1"),
					},
				},
			},
			DetachGroupPolicyOutput: iam.DetachGroupPolicyOutput{},
			ListGroupPoliciesOutput: iam.ListGroupPoliciesOutput{
				PolicyNames: []*string{
					awsgo.String("policy2"),
				},
			},
			DeleteGroupPolicyOutput: iam.DeleteGroupPolicyOutput{},
			DeleteGroupOutput:       iam.DeleteGroupOutput{},
		},
	}

	err := ig.nukeAll([]*string{awsgo.String("group1")})
	require.NoError(t, err)
}

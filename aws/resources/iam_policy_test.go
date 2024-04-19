package resources

import (
	"context"
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

type mockedIAMPolicies struct {
	iamiface.IAMAPI
	ListPoliciesPagesOutput iam.ListPoliciesOutput

	ListEntitiesForPolicyPagesOutput iam.ListEntitiesForPolicyOutput
	DetachUserPolicyOutput           iam.DetachUserPolicyOutput
	DetachGroupPolicyOutput          iam.DetachGroupPolicyOutput
	DetachRolePolicyOutput           iam.DetachRolePolicyOutput
	ListPolicyVersionsPagesOutput    iam.ListPolicyVersionsOutput
	DeletePolicyVersionOutput        iam.DeletePolicyVersionOutput
	DeletePolicyOutput               iam.DeletePolicyOutput
}

func (m mockedIAMPolicies) ListPoliciesPages(input *iam.ListPoliciesInput, fn func(*iam.ListPoliciesOutput, bool) bool) error {
	fn(&m.ListPoliciesPagesOutput, true)
	return nil
}

func (m mockedIAMPolicies) ListEntitiesForPolicyPages(input *iam.ListEntitiesForPolicyInput, fn func(*iam.ListEntitiesForPolicyOutput, bool) bool) error {
	fn(&m.ListEntitiesForPolicyPagesOutput, true)
	return nil
}

func (m mockedIAMPolicies) DetachUserPolicy(input *iam.DetachUserPolicyInput) (*iam.DetachUserPolicyOutput, error) {
	return &m.DetachUserPolicyOutput, nil
}

func (m mockedIAMPolicies) DetachGroupPolicy(input *iam.DetachGroupPolicyInput) (*iam.DetachGroupPolicyOutput, error) {
	return &m.DetachGroupPolicyOutput, nil
}

func (m mockedIAMPolicies) DetachRolePolicy(input *iam.DetachRolePolicyInput) (*iam.DetachRolePolicyOutput, error) {
	return &m.DetachRolePolicyOutput, nil
}

func (m mockedIAMPolicies) ListPolicyVersionsPages(input *iam.ListPolicyVersionsInput, fn func(*iam.ListPolicyVersionsOutput, bool) bool) error {
	fn(&m.ListPolicyVersionsPagesOutput, true)
	return nil
}

func (m mockedIAMPolicies) DeletePolicyVersion(input *iam.DeletePolicyVersionInput) (*iam.DeletePolicyVersionOutput, error) {
	return &m.DeletePolicyVersionOutput, nil
}

func (m mockedIAMPolicies) DeletePolicy(input *iam.DeletePolicyInput) (*iam.DeletePolicyOutput, error) {
	return &m.DeletePolicyOutput, nil
}

func TestIAMPolicy_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "MyPolicy1"
	testName2 := "MyPolicy2"
	testArn1 := "arn:aws:iam::123456789012:policy/MyPolicy1"
	testArn2 := "arn:aws:iam::123456789012:policy/MyPolicy2"
	now := time.Now()
	ip := IAMPolicies{
		Client: mockedIAMPolicies{
			ListPoliciesPagesOutput: iam.ListPoliciesOutput{
				Policies: []*iam.Policy{
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
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ip.getAll(context.Background(), config.Config{
				IAMPolicies: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestIAMPolicy_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	ip := IAMPolicies{
		Client: mockedIAMPolicies{
			ListEntitiesForPolicyPagesOutput: iam.ListEntitiesForPolicyOutput{
				PolicyGroups: []*iam.PolicyGroup{
					{GroupName: aws.String("group1")},
				},
				PolicyUsers: []*iam.PolicyUser{
					{UserName: aws.String("user1")},
				},
				PolicyRoles: []*iam.PolicyRole{
					{RoleName: aws.String("role1")},
				},
			},
			DetachUserPolicyOutput:  iam.DetachUserPolicyOutput{},
			DetachGroupPolicyOutput: iam.DetachGroupPolicyOutput{},
			DetachRolePolicyOutput:  iam.DetachRolePolicyOutput{},
			ListPolicyVersionsPagesOutput: iam.ListPolicyVersionsOutput{
				Versions: []*iam.PolicyVersion{
					{
						VersionId:        aws.String("v1"),
						IsDefaultVersion: aws.Bool(false),
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

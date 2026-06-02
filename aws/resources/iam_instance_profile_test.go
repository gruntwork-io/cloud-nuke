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

type mockedIAMInstanceProfiles struct {
	IAMInstanceProfilesAPI

	GetInstanceProfileOutput            iam.GetInstanceProfileOutput
	ListInstanceProfilesOutput          iam.ListInstanceProfilesOutput
	ListInstanceProfileTagsOutput       map[string]iam.ListInstanceProfileTagsOutput
	RemoveRoleFromInstanceProfileOutput iam.RemoveRoleFromInstanceProfileOutput
	DeleteInstanceProfileOutput         iam.DeleteInstanceProfileOutput
}

func (m mockedIAMInstanceProfiles) ListInstanceProfileTags(ctx context.Context, params *iam.ListInstanceProfileTagsInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfileTagsOutput, error) {
	out := m.ListInstanceProfileTagsOutput[aws.ToString(params.InstanceProfileName)]
	return &out, nil
}

func (m mockedIAMInstanceProfiles) GetInstanceProfile(ctx context.Context, params *iam.GetInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.GetInstanceProfileOutput, error) {
	return &m.GetInstanceProfileOutput, nil
}
func (m mockedIAMInstanceProfiles) ListInstanceProfiles(ctx context.Context, params *iam.ListInstanceProfilesInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfilesOutput, error) {
	return &m.ListInstanceProfilesOutput, nil
}
func (m mockedIAMInstanceProfiles) RemoveRoleFromInstanceProfile(ctx context.Context, params *iam.RemoveRoleFromInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.RemoveRoleFromInstanceProfileOutput, error) {
	return &m.RemoveRoleFromInstanceProfileOutput, nil
}
func (m mockedIAMInstanceProfiles) DeleteInstanceProfile(ctx context.Context, params *iam.DeleteInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.DeleteInstanceProfileOutput, error) {
	return &m.DeleteInstanceProfileOutput, nil
}

func TestIAMInstanceProfiles_ListIAMInstanceProfiles(t *testing.T) {
	t.Parallel()
	testName1 := "MyInstanceProfiles1"
	testName2 := "MyInstanceProfiles2"
	testArn1 := "arn:aws:iam::123456789012:instance-profile/MyInstanceProfiles1"
	testArn2 := "arn:aws:iam::123456789012:instance-profile/MyInstanceProfiles2"
	now := time.Now()

	testName3 := "MyInstanceProfiles3"
	testArn3 := "arn:aws:iam::123456789012:instance-profile/MyInstanceProfiles3"

	client := mockedIAMInstanceProfiles{
		ListInstanceProfilesOutput: iam.ListInstanceProfilesOutput{
			InstanceProfiles: []types.InstanceProfile{
				{
					Arn:                 aws.String(testArn1),
					InstanceProfileName: aws.String(testName1),
					CreateDate:          aws.Time(now),
				},
				{
					Arn:                 aws.String(testArn2),
					InstanceProfileName: aws.String(testName2),
					CreateDate:          aws.Time(now.Add(1)),
				},
				{
					Arn:                 aws.String(testArn3),
					InstanceProfileName: aws.String(testName3),
					CreateDate:          aws.Time(now.Add(2)),
				},
			},
		},
		// The ListInstanceProfiles API does not return tags; they are fetched separately
		// via ListInstanceProfileTags. testName3 carries the cloud-nuke-excluded tag.
		ListInstanceProfileTagsOutput: map[string]iam.ListInstanceProfileTagsOutput{
			testName1: {Tags: []types.Tag{{Key: aws.String("somearn"), Value: aws.String("some" + testArn1)}}},
			testName2: {Tags: []types.Tag{{Key: aws.String("somearn"), Value: aws.String("some" + testArn2)}}},
			testName3: {Tags: []types.Tag{{Key: aws.String("cloud-nuke-excluded"), Value: aws.String("true")}}},
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
		// Regression test for #1140: a profile tagged cloud-nuke-excluded=true must be
		// filtered out even with no explicit filters. This only works because tags are
		// fetched via ListInstanceProfileTags; ListInstanceProfiles returns no tags.
		"defaultExclusionTag": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("^" + testName1 + "$"),
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
			names, err := listIAMInstanceProfiles(context.Background(), client, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestIAMInstanceProfiles_DeleteIAMInstanceProfile(t *testing.T) {
	t.Parallel()

	client := mockedIAMInstanceProfiles{
		GetInstanceProfileOutput: iam.GetInstanceProfileOutput{
			InstanceProfile: &types.InstanceProfile{
				InstanceProfileName: aws.String("profile1"),
				Roles: []types.Role{
					{
						RoleName: aws.String("role1"),
					},
					{
						RoleName: aws.String("role2"),
					},
				},
			},
		},
		RemoveRoleFromInstanceProfileOutput: iam.RemoveRoleFromInstanceProfileOutput{},
		DeleteInstanceProfileOutput:         iam.DeleteInstanceProfileOutput{},
	}

	err := deleteIAMInstanceProfile(context.Background(), client, aws.String("profile1"))
	require.NoError(t, err)
}

func TestIAMInstanceProfiles_DeleteIAMInstanceProfile_NoRoles(t *testing.T) {
	t.Parallel()

	client := mockedIAMInstanceProfiles{
		GetInstanceProfileOutput: iam.GetInstanceProfileOutput{
			InstanceProfile: &types.InstanceProfile{
				InstanceProfileName: aws.String("profile-no-roles"),
				Roles:               []types.Role{},
			},
		},
		DeleteInstanceProfileOutput: iam.DeleteInstanceProfileOutput{},
	}

	err := deleteIAMInstanceProfile(context.Background(), client, aws.String("profile-no-roles"))
	require.NoError(t, err)
}

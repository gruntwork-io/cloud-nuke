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

type mockedIAMServiceLinkedRoles struct {
	IAMServiceLinkedRolesAPI
	ListRolesOutput                          iam.ListRolesOutput
	DeleteServiceLinkedRoleOutput            iam.DeleteServiceLinkedRoleOutput
	GetServiceLinkedRoleDeletionStatusOutput iam.GetServiceLinkedRoleDeletionStatusOutput
}

func (m mockedIAMServiceLinkedRoles) ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error) {
	return &m.ListRolesOutput, nil
}

func (m mockedIAMServiceLinkedRoles) DeleteServiceLinkedRole(ctx context.Context, params *iam.DeleteServiceLinkedRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteServiceLinkedRoleOutput, error) {
	return &m.DeleteServiceLinkedRoleOutput, nil
}

func (m mockedIAMServiceLinkedRoles) GetServiceLinkedRoleDeletionStatus(ctx context.Context, params *iam.GetServiceLinkedRoleDeletionStatusInput, optFns ...func(*iam.Options)) (*iam.GetServiceLinkedRoleDeletionStatusOutput, error) {
	return &m.GetServiceLinkedRoleDeletionStatusOutput, nil
}

func TestIAMServiceLinkedRoles_GetAll(t *testing.T) {
	t.Parallel()
	now := time.Now()
	testName1 := "test-role1"
	testName2 := "test-role2"
	islr := IAMServiceLinkedRoles{
		Client: &mockedIAMServiceLinkedRoles{
			ListRolesOutput: iam.ListRolesOutput{
				Roles: []types.Role{
					{
						RoleName:   aws.String(testName1),
						CreateDate: aws.Time(now),
						Arn:        aws.String("aws-service-role"),
					},
					{
						RoleName:   aws.String(testName2),
						CreateDate: aws.Time(now.Add(1)),
						Arn:        aws.String("aws-service-role"),
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
			names, err := islr.getAll(context.Background(), config.Config{
				IAMServiceLinkedRoles: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestIAMServiceLinkedRoles_NukeAll(t *testing.T) {
	t.Parallel()
	islr := IAMServiceLinkedRoles{
		Client: &mockedIAMServiceLinkedRoles{
			DeleteServiceLinkedRoleOutput: iam.DeleteServiceLinkedRoleOutput{},
			GetServiceLinkedRoleDeletionStatusOutput: iam.GetServiceLinkedRoleDeletionStatusOutput{
				Status: types.DeletionTaskStatusTypeSucceeded,
			},
		},
	}

	err := islr.nukeAll([]*string{aws.String("test-role1")})
	require.NoError(t, err)
}

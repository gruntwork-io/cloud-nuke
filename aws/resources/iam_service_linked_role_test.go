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

type mockedIAMServiceLinkedRoles struct {
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

func TestIAMServiceLinkedRoles_List(t *testing.T) {
	t.Parallel()
	now := time.Now()
	testName1 := "test-role1"
	testName2 := "test-role2"
	client := mockedIAMServiceLinkedRoles{
		ListRolesOutput: iam.ListRolesOutput{
			Roles: []types.Role{
				{
					RoleName:   aws.String(testName1),
					CreateDate: aws.Time(now),
					Arn:        aws.String("arn:aws:iam::123456789012:role/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling"),
				},
				{
					RoleName:   aws.String(testName2),
					CreateDate: aws.Time(now.Add(1)),
					Arn:        aws.String("arn:aws:iam::123456789012:role/aws-service-role/eks.amazonaws.com/AWSServiceRoleForEKS"),
				},
				{
					// This role should be filtered out (not a service-linked role)
					RoleName:   aws.String("regular-role"),
					CreateDate: aws.Time(now),
					Arn:        aws.String("arn:aws:iam::123456789012:role/regular-role"),
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
			names, err := listIAMServiceLinkedRoles(context.Background(), client, resource.Scope{Region: "global"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestIAMServiceLinkedRoles_Delete(t *testing.T) {
	t.Parallel()
	client := mockedIAMServiceLinkedRoles{
		DeleteServiceLinkedRoleOutput: iam.DeleteServiceLinkedRoleOutput{
			DeletionTaskId: aws.String("task/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling/test-task-id"),
		},
		GetServiceLinkedRoleDeletionStatusOutput: iam.GetServiceLinkedRoleDeletionStatusOutput{
			Status: types.DeletionTaskStatusTypeSucceeded,
		},
	}

	err := deleteIAMServiceLinkedRole(context.Background(), client, aws.String("test-role1"))
	require.NoError(t, err)
}

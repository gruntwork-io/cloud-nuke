package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/stretchr/testify/require"
)

type mockedIAMServiceLinkedRoles struct {
	iamiface.IAMAPI
	ListRolesPagesOutput                     iam.ListRolesOutput
	DeleteServiceLinkedRoleOutput            iam.DeleteServiceLinkedRoleOutput
	GetServiceLinkedRoleDeletionStatusOutput iam.GetServiceLinkedRoleDeletionStatusOutput
}

func (m mockedIAMServiceLinkedRoles) ListRolesPages(input *iam.ListRolesInput, fn func(*iam.ListRolesOutput, bool) bool) error {
	fn(&m.ListRolesPagesOutput, true)
	return nil
}

func (m mockedIAMServiceLinkedRoles) DeleteServiceLinkedRole(input *iam.DeleteServiceLinkedRoleInput) (*iam.DeleteServiceLinkedRoleOutput, error) {
	return &m.DeleteServiceLinkedRoleOutput, nil
}

func (m mockedIAMServiceLinkedRoles) GetServiceLinkedRoleDeletionStatus(input *iam.GetServiceLinkedRoleDeletionStatusInput) (*iam.GetServiceLinkedRoleDeletionStatusOutput, error) {
	return &m.GetServiceLinkedRoleDeletionStatusOutput, nil
}

func TestIAMServiceLinkedRoles_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	now := time.Now()
	testName1 := "test-role1"
	testName2 := "test-role2"
	islr := IAMServiceLinkedRoles{
		Client: &mockedIAMServiceLinkedRoles{
			ListRolesPagesOutput: iam.ListRolesOutput{
				Roles: []*iam.Role{
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestIAMServiceLinkedRoles_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	islr := IAMServiceLinkedRoles{

		Client: &mockedIAMServiceLinkedRoles{
			DeleteServiceLinkedRoleOutput: iam.DeleteServiceLinkedRoleOutput{},
			GetServiceLinkedRoleDeletionStatusOutput: iam.GetServiceLinkedRoleDeletionStatusOutput{
				Status: aws.String("SUCCEEDED"),
			},
		},
	}

	err := islr.nukeAll([]*string{aws.String("test-role1")})
	require.NoError(t, err)
}

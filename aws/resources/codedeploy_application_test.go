package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedCodeDeployApplications struct {
	CodeDeployApplicationsAPI
	ListApplicationsOutput     codedeploy.ListApplicationsOutput
	BatchGetApplicationsOutput codedeploy.BatchGetApplicationsOutput
	DeleteApplicationOutput    codedeploy.DeleteApplicationOutput
}

func (m mockedCodeDeployApplications) ListApplications(ctx context.Context, params *codedeploy.ListApplicationsInput, optFns ...func(*codedeploy.Options)) (*codedeploy.ListApplicationsOutput, error) {
	return &m.ListApplicationsOutput, nil
}

func (m mockedCodeDeployApplications) BatchGetApplications(ctx context.Context, input *codedeploy.BatchGetApplicationsInput, optFns ...func(*codedeploy.Options)) (*codedeploy.BatchGetApplicationsOutput, error) {
	// Filter out applications that don't match the input names
	names := make(map[string]bool)
	for _, name := range input.ApplicationNames {
		names[name] = true
	}

	var matched []types.ApplicationInfo
	for _, info := range m.BatchGetApplicationsOutput.ApplicationsInfo {
		if names[*info.ApplicationName] {
			matched = append(matched, info)
		}
	}

	return &codedeploy.BatchGetApplicationsOutput{
		ApplicationsInfo: matched,
	}, nil
}

func (m mockedCodeDeployApplications) DeleteApplication(ctx context.Context, params *codedeploy.DeleteApplicationInput, optFns ...func(*codedeploy.Options)) (*codedeploy.DeleteApplicationOutput, error) {
	return &m.DeleteApplicationOutput, nil
}

func TestCodeDeployApplication_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "cloud-nuke-test-1"
	testName2 := "cloud-nuke-test-2"
	now := time.Now()
	c := CodeDeployApplications{
		Client: mockedCodeDeployApplications{
			ListApplicationsOutput: codedeploy.ListApplicationsOutput{
				Applications: []string{
					testName1,
					testName2,
				},
			},
			BatchGetApplicationsOutput: codedeploy.BatchGetApplicationsOutput{
				ApplicationsInfo: []types.ApplicationInfo{
					{
						ApplicationName: aws.String(testName1),
						CreateTime:      aws.Time(now),
					},
					{
						ApplicationName: aws.String(testName2),
						CreateTime:      aws.Time(now.Add(1)),
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
					TimeAfter: aws.Time(now.Add(-1)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := c.getAll(context.Background(), config.Config{
				CodeDeployApplications: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestCodeDeployApplication_NukeAll(t *testing.T) {
	c := CodeDeployApplications{
		Client: mockedCodeDeployApplications{
			DeleteApplicationOutput: codedeploy.DeleteApplicationOutput{},
		},
	}

	err := c.nukeAll([]string{"test"})
	require.NoError(t, err)
}

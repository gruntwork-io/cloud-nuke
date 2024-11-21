package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedLaunchTemplate struct {
	LaunchTemplatesAPI
	DescribeLaunchTemplatesOutput ec2.DescribeLaunchTemplatesOutput
	DeleteLaunchTemplateOutput    ec2.DeleteLaunchTemplateOutput
}

func (m mockedLaunchTemplate) DescribeLaunchTemplates(ctx context.Context, params *ec2.DescribeLaunchTemplatesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeLaunchTemplatesOutput, error) {
	return &m.DescribeLaunchTemplatesOutput, nil
}

func (m mockedLaunchTemplate) DeleteLaunchTemplate(ctx context.Context, params *ec2.DeleteLaunchTemplateInput, optFns ...func(*ec2.Options)) (*ec2.DeleteLaunchTemplateOutput, error) {
	return &m.DeleteLaunchTemplateOutput, nil
}

func TestLaunchTemplate_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	testName1 := "test-launch-template1"
	testName2 := "test-launch-template2"
	lt := LaunchTemplates{
		Client: mockedLaunchTemplate{
			DescribeLaunchTemplatesOutput: ec2.DescribeLaunchTemplatesOutput{
				LaunchTemplates: []types.LaunchTemplate{
					{
						LaunchTemplateName: aws.String(testName1),
						CreateTime:         aws.Time(now),
					},
					{
						LaunchTemplateName: aws.String(testName2),
						CreateTime:         aws.Time(now.Add(1)),
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
			names, err := lt.getAll(context.Background(), config.Config{
				LaunchTemplate: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestLaunchTemplate_NukeAll(t *testing.T) {

	t.Parallel()

	lt := LaunchTemplates{
		Client: mockedLaunchTemplate{
			DeleteLaunchTemplateOutput: ec2.DeleteLaunchTemplateOutput{},
		},
	}

	err := lt.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

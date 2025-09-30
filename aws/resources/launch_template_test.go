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
	testId1 := "lt-123456789"
	testId2 := "lt-987654321"
	templateWithTags := createLaunchTemplateWithTags(testName1, testId1, now)
	templateWithoutTags := createLaunchTemplateWithoutTags(testName2, testId2, now.Add(1))

	lt := LaunchTemplates{
		Client: mockedLaunchTemplate{
			DescribeLaunchTemplatesOutput: ec2.DescribeLaunchTemplatesOutput{
				LaunchTemplates: []types.LaunchTemplate{
					templateWithTags,
					templateWithoutTags,
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
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
					},
				},
			},
			expected: []string{testName2},
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

func createLaunchTemplateWithTags(name, id string, createTime time.Time) types.LaunchTemplate {
	return types.LaunchTemplate{
		LaunchTemplateName: aws.String(name),
		LaunchTemplateId:   aws.String(id),
		CreateTime:         aws.Time(createTime),
		Tags: []types.Tag{
			{
				Key:   aws.String("Environment"),
				Value: aws.String("test"),
			},
		},
	}
}

func createLaunchTemplateWithoutTags(name, id string, createTime time.Time) types.LaunchTemplate {
	return types.LaunchTemplate{
		LaunchTemplateName: aws.String(name),
		LaunchTemplateId:   aws.String(id),
		CreateTime:         aws.Time(createTime),
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

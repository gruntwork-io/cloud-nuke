package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedLaunchTemplate struct {
	ec2iface.EC2API
	DescribeLaunchTemplatesOutput ec2.DescribeLaunchTemplatesOutput
	DeleteLaunchTemplateOutput    ec2.DeleteLaunchTemplateOutput
}

func (m mockedLaunchTemplate) DescribeLaunchTemplates(input *ec2.DescribeLaunchTemplatesInput) (*ec2.DescribeLaunchTemplatesOutput, error) {
	return &m.DescribeLaunchTemplatesOutput, nil
}

func (m mockedLaunchTemplate) DeleteLaunchTemplate(input *ec2.DeleteLaunchTemplateInput) (*ec2.DeleteLaunchTemplateOutput, error) {
	return &m.DeleteLaunchTemplateOutput, nil
}

func TestLaunchTemplate_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	now := time.Now()
	testName1 := "test-launch-template1"
	testName2 := "test-launch-template2"
	lt := LaunchTemplates{
		Client: mockedLaunchTemplate{
			DescribeLaunchTemplatesOutput: ec2.DescribeLaunchTemplatesOutput{
				LaunchTemplates: []*ec2.LaunchTemplate{
					{
						LaunchTemplateName: awsgo.String(testName1),
						CreateTime:         awsgo.Time(now),
					},
					{
						LaunchTemplateName: awsgo.String(testName2),
						CreateTime:         awsgo.Time(now.Add(1)),
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
			names, err := lt.getAll(config.Config{
				LaunchTemplate: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestLaunchTemplate_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	lt := LaunchTemplates{
		Client: mockedLaunchTemplate{
			DeleteLaunchTemplateOutput: ec2.DeleteLaunchTemplateOutput{},
		},
	}

	err := lt.nukeAll([]*string{awsgo.String("test")})
	require.NoError(t, err)
}

package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedLaunchTemplate struct {
	ec2iface.EC2API
	DescribeLaunchTemplatesOutput ec2.DescribeLaunchTemplatesOutput
	DeleteLaunchTemplateOutput    ec2.DeleteLaunchTemplateOutput
}

func (m mockedLaunchTemplate) DescribeLaunchTemplatesWithContext(_ aws.Context, input *ec2.DescribeLaunchTemplatesInput, _ ...request.Option) (*ec2.DescribeLaunchTemplatesOutput, error) {
	return &m.DescribeLaunchTemplatesOutput, nil
}

func (m mockedLaunchTemplate) DeleteLaunchTemplateWithContext(_ aws.Context, input *ec2.DeleteLaunchTemplateInput, _ ...request.Option) (*ec2.DeleteLaunchTemplateOutput, error) {
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
			names, err := lt.getAll(context.Background(), config.Config{
				LaunchTemplate: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
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

	err := lt.nukeAll([]*string{awsgo.String("test")})
	require.NoError(t, err)
}

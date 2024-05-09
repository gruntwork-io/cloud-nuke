package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

type mockedLaunchConfiguration struct {
	autoscalingiface.AutoScalingAPI
	DescribeLaunchConfigurationsOutput autoscaling.DescribeLaunchConfigurationsOutput
	DeleteLaunchConfigurationOutput    autoscaling.DeleteLaunchConfigurationOutput
}

func (m mockedLaunchConfiguration) DescribeLaunchConfigurationsWithContext(_ aws.Context, input *autoscaling.DescribeLaunchConfigurationsInput, _ ...request.Option) (*autoscaling.DescribeLaunchConfigurationsOutput, error) {
	return &m.DescribeLaunchConfigurationsOutput, nil
}

func (m mockedLaunchConfiguration) DeleteLaunchConfigurationWithContext(_ aws.Context, input *autoscaling.DeleteLaunchConfigurationInput, _ ...request.Option) (*autoscaling.DeleteLaunchConfigurationOutput, error) {
	return &m.DeleteLaunchConfigurationOutput, nil
}

func TestLaunchConfigurations_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-launch-config1"
	testName2 := "test-launch-config2"
	now := time.Now()
	lc := LaunchConfigs{
		Client: mockedLaunchConfiguration{
			DescribeLaunchConfigurationsOutput: autoscaling.DescribeLaunchConfigurationsOutput{
				LaunchConfigurations: []*autoscaling.LaunchConfiguration{
					{
						LaunchConfigurationName: awsgo.String(testName1),
						CreatedTime:             awsgo.Time(now),
					},
					{
						LaunchConfigurationName: awsgo.String(testName2),
						CreatedTime:             awsgo.Time(now.Add(1)),
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
					TimeAfter: awsgo.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := lc.getAll(context.Background(), config.Config{
				LaunchConfiguration: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestLaunchConfigurations_NukeAll(t *testing.T) {

	t.Parallel()

	lc := LaunchConfigs{
		Client: mockedLaunchConfiguration{
			DeleteLaunchConfigurationOutput: autoscaling.DeleteLaunchConfigurationOutput{},
		},
	}

	err := lc.nukeAll([]*string{awsgo.String("test")})
	require.NoError(t, err)
}

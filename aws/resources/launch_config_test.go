package resources

import (
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

type mockedLaunchConfiguration struct {
	autoscalingiface.AutoScalingAPI
	DescribeLaunchConfigurationsOutput autoscaling.DescribeLaunchConfigurationsOutput
	DeleteLaunchConfigurationOutput    autoscaling.DeleteLaunchConfigurationOutput
}

func (m mockedLaunchConfiguration) DescribeLaunchConfigurations(input *autoscaling.DescribeLaunchConfigurationsInput) (*autoscaling.DescribeLaunchConfigurationsOutput, error) {
	return &m.DescribeLaunchConfigurationsOutput, nil
}

func (m mockedLaunchConfiguration) DeleteLaunchConfiguration(input *autoscaling.DeleteLaunchConfigurationInput) (*autoscaling.DeleteLaunchConfigurationOutput, error) {
	return &m.DeleteLaunchConfigurationOutput, nil
}

func TestLaunchConfigurations_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
			names, err := lc.getAll(config.Config{
				LaunchConfiguration: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestLaunchConfigurations_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	lc := LaunchConfigs{
		Client: mockedLaunchConfiguration{
			DeleteLaunchConfigurationOutput: autoscaling.DeleteLaunchConfigurationOutput{},
		},
	}

	err := lc.nukeAll([]*string{awsgo.String("test")})
	require.NoError(t, err)
}

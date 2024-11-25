package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedLaunchConfiguration struct {
	LaunchConfigsAPI
	DescribeLaunchConfigurationsOutput autoscaling.DescribeLaunchConfigurationsOutput
	DeleteLaunchConfigurationOutput    autoscaling.DeleteLaunchConfigurationOutput
}

func (m mockedLaunchConfiguration) DescribeLaunchConfigurations(ctx context.Context, params *autoscaling.DescribeLaunchConfigurationsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeLaunchConfigurationsOutput, error) {
	return &m.DescribeLaunchConfigurationsOutput, nil
}

func (m mockedLaunchConfiguration) DeleteLaunchConfiguration(ctx context.Context, params *autoscaling.DeleteLaunchConfigurationInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DeleteLaunchConfigurationOutput, error) {
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
				LaunchConfigurations: []types.LaunchConfiguration{
					{
						LaunchConfigurationName: aws.String(testName1),
						CreatedTime:             aws.Time(now),
					},
					{
						LaunchConfigurationName: aws.String(testName2),
						CreatedTime:             aws.Time(now.Add(1)),
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
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
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

	err := lc.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

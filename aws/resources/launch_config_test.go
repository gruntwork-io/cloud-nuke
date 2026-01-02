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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

// mockLaunchConfigsClient implements LaunchConfigsAPI for testing.
type mockLaunchConfigsClient struct {
	DescribeLaunchConfigurationsOutput autoscaling.DescribeLaunchConfigurationsOutput
	DeleteLaunchConfigurationOutput    autoscaling.DeleteLaunchConfigurationOutput
}

func (m *mockLaunchConfigsClient) DescribeLaunchConfigurations(ctx context.Context, params *autoscaling.DescribeLaunchConfigurationsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeLaunchConfigurationsOutput, error) {
	return &m.DescribeLaunchConfigurationsOutput, nil
}

func (m *mockLaunchConfigsClient) DeleteLaunchConfiguration(ctx context.Context, params *autoscaling.DeleteLaunchConfigurationInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DeleteLaunchConfigurationOutput, error) {
	return &m.DeleteLaunchConfigurationOutput, nil
}

func TestLaunchConfigs_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-lc-1"
	testName2 := "test-lc-2"
	now := time.Now()

	mock := &mockLaunchConfigsClient{
		DescribeLaunchConfigurationsOutput: autoscaling.DescribeLaunchConfigurationsOutput{
			LaunchConfigurations: []types.LaunchConfiguration{
				{
					LaunchConfigurationName: aws.String(testName1),
					CreatedTime:             aws.Time(now),
				},
				{
					LaunchConfigurationName: aws.String(testName2),
					CreatedTime:             aws.Time(now.Add(1 * time.Hour)),
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
						RE: *regexp.MustCompile("test-lc-1"),
					}},
				},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listLaunchConfigs(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestLaunchConfigs_NukeAll(t *testing.T) {
	t.Parallel()

	mock := &mockLaunchConfigsClient{
		DeleteLaunchConfigurationOutput: autoscaling.DeleteLaunchConfigurationOutput{},
	}

	err := deleteLaunchConfig(context.Background(), mock, aws.String("test-lc"))
	require.NoError(t, err)
}

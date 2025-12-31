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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestLaunchConfigs_ResourceName(t *testing.T) {
	r := NewLaunchConfigs()
	assert.Equal(t, "lc", r.ResourceName())
}

func TestLaunchConfigs_MaxBatchSize(t *testing.T) {
	r := NewLaunchConfigs()
	assert.Equal(t, 49, r.MaxBatchSize())
}

func TestListLaunchConfigs(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockLaunchConfigsClient{
		DescribeLaunchConfigurationsOutput: autoscaling.DescribeLaunchConfigurationsOutput{
			LaunchConfigurations: []types.LaunchConfiguration{
				{LaunchConfigurationName: aws.String("lc1"), CreatedTime: aws.Time(now)},
				{LaunchConfigurationName: aws.String("lc2"), CreatedTime: aws.Time(now)},
			},
		},
	}

	names, err := listLaunchConfigs(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"lc1", "lc2"}, aws.ToStringSlice(names))
}

func TestListLaunchConfigs_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockLaunchConfigsClient{
		DescribeLaunchConfigurationsOutput: autoscaling.DescribeLaunchConfigurationsOutput{
			LaunchConfigurations: []types.LaunchConfiguration{
				{LaunchConfigurationName: aws.String("lc1"), CreatedTime: aws.Time(now)},
				{LaunchConfigurationName: aws.String("skip-this"), CreatedTime: aws.Time(now)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listLaunchConfigs(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"lc1"}, aws.ToStringSlice(names))
}

func TestDeleteLaunchConfig(t *testing.T) {
	t.Parallel()

	mock := &mockLaunchConfigsClient{}
	err := deleteLaunchConfig(context.Background(), mock, aws.String("test-lc"))
	require.NoError(t, err)
}

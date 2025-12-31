package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockConfigServiceRecordersClient struct {
	DescribeConfigurationRecordersOutput configservice.DescribeConfigurationRecordersOutput
	DeleteConfigurationRecorderOutput    configservice.DeleteConfigurationRecorderOutput
}

func (m *mockConfigServiceRecordersClient) DescribeConfigurationRecorders(ctx context.Context, params *configservice.DescribeConfigurationRecordersInput, optFns ...func(*configservice.Options)) (*configservice.DescribeConfigurationRecordersOutput, error) {
	return &m.DescribeConfigurationRecordersOutput, nil
}

func (m *mockConfigServiceRecordersClient) DeleteConfigurationRecorder(ctx context.Context, params *configservice.DeleteConfigurationRecorderInput, optFns ...func(*configservice.Options)) (*configservice.DeleteConfigurationRecorderOutput, error) {
	return &m.DeleteConfigurationRecorderOutput, nil
}

func TestListConfigServiceRecorders(t *testing.T) {
	t.Parallel()

	mock := &mockConfigServiceRecordersClient{
		DescribeConfigurationRecordersOutput: configservice.DescribeConfigurationRecordersOutput{
			ConfigurationRecorders: []types.ConfigurationRecorder{
				{Name: aws.String("test-recorder-1")},
				{Name: aws.String("test-recorder-2")},
			},
		},
	}

	names, err := listConfigServiceRecorders(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"test-recorder-1", "test-recorder-2"}, aws.ToStringSlice(names))
}

func TestListConfigServiceRecorders_WithFilter(t *testing.T) {
	t.Parallel()

	mock := &mockConfigServiceRecordersClient{
		DescribeConfigurationRecordersOutput: configservice.DescribeConfigurationRecordersOutput{
			ConfigurationRecorders: []types.ConfigurationRecorder{
				{Name: aws.String("test-recorder-1")},
				{Name: aws.String("test-recorder-2")},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-recorder-1")}},
		},
	}

	names, err := listConfigServiceRecorders(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"test-recorder-2"}, aws.ToStringSlice(names))
}

func TestDeleteConfigServiceRecorder(t *testing.T) {
	t.Parallel()

	mock := &mockConfigServiceRecordersClient{}
	err := deleteConfigServiceRecorder(context.Background(), mock, aws.String("test-recorder"))
	require.NoError(t, err)
}

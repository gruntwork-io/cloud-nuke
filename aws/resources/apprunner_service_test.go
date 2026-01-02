package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/aws/aws-sdk-go-v2/service/apprunner/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockAppRunnerServiceClient struct {
	DeleteServiceOutput apprunner.DeleteServiceOutput
	ListServicesOutput  apprunner.ListServicesOutput
}

func (m *mockAppRunnerServiceClient) DeleteService(ctx context.Context, params *apprunner.DeleteServiceInput, optFns ...func(*apprunner.Options)) (*apprunner.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func (m *mockAppRunnerServiceClient) ListServices(ctx context.Context, params *apprunner.ListServicesInput, optFns ...func(*apprunner.Options)) (*apprunner.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func TestListAppRunnerServices(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockAppRunnerServiceClient{
		ListServicesOutput: apprunner.ListServicesOutput{
			ServiceSummaryList: []types.ServiceSummary{
				{ServiceName: aws.String("svc1"), ServiceArn: aws.String("arn::svc1"), CreatedAt: aws.Time(now)},
				{ServiceName: aws.String("svc2"), ServiceArn: aws.String("arn::svc2"), CreatedAt: aws.Time(now.Add(1 * time.Hour))},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{"arn::svc1", "arn::svc2"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("svc1")}},
				},
			},
			expected: []string{"arn::svc2"},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{"arn::svc1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			arns, err := listAppRunnerServices(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(arns))
		})
	}
}

func TestDeleteAppRunnerService(t *testing.T) {
	t.Parallel()
	err := deleteAppRunnerService(context.Background(), &mockAppRunnerServiceClient{}, aws.String("arn::test"))
	require.NoError(t, err)
}

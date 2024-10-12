package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/aws/aws-sdk-go-v2/service/apprunner/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedAppRunnerService struct {
	AppRunnerServiceAPI
	DeleteServiceOutput apprunner.DeleteServiceOutput
	ListServicesOutput  apprunner.ListServicesOutput
}

func (m mockedAppRunnerService) DeleteService(ctx context.Context, params *apprunner.DeleteServiceInput, optFns ...func(*apprunner.Options)) (*apprunner.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func (m mockedAppRunnerService) ListServices(ctx context.Context, params *apprunner.ListServicesInput, optFns ...func(*apprunner.Options)) (*apprunner.ListServicesOutput, error) {
	return &m.ListServicesOutput, nil
}

func Test_AppRunnerService_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-service-1"
	testName2 := "test-service-2"
	now := time.Now()
	service := AppRunnerService{
		Client: mockedAppRunnerService{
			ListServicesOutput: apprunner.ListServicesOutput{
				ServiceSummaryList: []types.ServiceSummary{
					{
						ServiceName: &testName1,
						ServiceArn:  aws.String(fmt.Sprintf("arn::%s", testName1)),
						CreatedAt:   &now,
					},
					{
						ServiceName: &testName2,
						ServiceArn:  aws.String(fmt.Sprintf("arn::%s", testName2)),
						CreatedAt:   aws.Time(now.Add(1)),
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
			expected:  []string{fmt.Sprintf("arn::%s", testName1), fmt.Sprintf("arn::%s", testName2)},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}},
				}},
			expected: []string{fmt.Sprintf("arn::%s", testName2)},
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
			names, err := service.getAll(context.Background(), config.Config{
				AppRunnerService: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestAppRunnerService_NukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-app-runner-service"
	service := AppRunnerService{
		Client: mockedAppRunnerService{
			DeleteServiceOutput: apprunner.DeleteServiceOutput{},
		},
	}

	err := service.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}

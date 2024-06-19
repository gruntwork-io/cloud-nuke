package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/apprunner"
	"github.com/aws/aws-sdk-go/service/apprunner/apprunneriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedAppRunnerService struct {
	apprunneriface.AppRunnerAPI
	ListServicesOutput  apprunner.ListServicesOutput
	DeleteServiceOutput apprunner.DeleteServiceOutput
}

func (m mockedAppRunnerService) ListServicesPagesWithContext(_ aws.Context, _ *apprunner.ListServicesInput, callback func(*apprunner.ListServicesOutput, bool) bool, _ ...request.Option) error {
	callback(&m.ListServicesOutput, true)
	return nil
}

func (m mockedAppRunnerService) DeleteServiceWithContext(aws.Context, *apprunner.DeleteServiceInput, ...request.Option) (*apprunner.DeleteServiceOutput, error) {
	return &m.DeleteServiceOutput, nil
}

func Test_AppRunnerService_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-service-1"
	testName2 := "test-service-2"
	now := time.Now()
	service := AppRunnerService{
		Client: mockedAppRunnerService{
			ListServicesOutput: apprunner.ListServicesOutput{
				ServiceSummaryList: []*apprunner.ServiceSummary{
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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

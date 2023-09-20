package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedLambda struct {
	lambdaiface.LambdaAPI
	ListFunctionsOutput  lambda.ListFunctionsOutput
	DeleteFunctionOutput lambda.DeleteFunctionOutput
}

func (m mockedLambda) ListFunctionsPages(input *lambda.ListFunctionsInput, fn func(*lambda.ListFunctionsOutput, bool) bool) error {
	fn(&m.ListFunctionsOutput, true)
	return nil
}

func (m mockedLambda) DeleteFunction(input *lambda.DeleteFunctionInput) (*lambda.DeleteFunctionOutput, error) {
	return &m.DeleteFunctionOutput, nil
}

func TestLambdaFunction_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-lambda-function1"
	testName2 := "test-lambda-function2"
	testTime := time.Now()

	layout := "2006-01-02T15:04:05.000+0000"
	testTimeStr := "2023-07-28T12:34:56.789+0000"
	testTime, err := time.Parse(layout, testTimeStr)
	require.NoError(t, err)

	lf := LambdaFunctions{
		Client: mockedLambda{
			ListFunctionsOutput: lambda.ListFunctionsOutput{
				Functions: []*lambda.FunctionConfiguration{
					{
						FunctionName: aws.String(testName1),
						LastModified: aws.String(testTimeStr),
					},
					{
						FunctionName: aws.String(testName2),
						LastModified: aws.String(testTimeStr),
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
					TimeAfter: aws.Time(testTime.Add(-2 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := lf.getAll(context.Background(), config.Config{
				LambdaFunction: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestLambdaFunction_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	lf := LambdaFunctions{
		Client: mockedLambda{
			DeleteFunctionOutput: lambda.DeleteFunctionOutput{},
		},
	}

	err := lf.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedLambda struct {
	LambdaFunctionsAPI
	DeleteFunctionOutput lambda.DeleteFunctionOutput
	ListFunctionsOutput  lambda.ListFunctionsOutput
}

func (m mockedLambda) DeleteFunction(ctx context.Context, params *lambda.DeleteFunctionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteFunctionOutput, error) {
	return &m.DeleteFunctionOutput, nil
}

func (m mockedLambda) ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error) {
	return &m.ListFunctionsOutput, nil
}

func TestLambdaFunction_GetAll(t *testing.T) {
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
				Functions: []types.FunctionConfiguration{
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestLambdaFunction_NukeAll(t *testing.T) {
	t.Parallel()

	lf := LambdaFunctions{
		Client: mockedLambda{
			DeleteFunctionOutput: lambda.DeleteFunctionOutput{},
		},
	}

	err := lf.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

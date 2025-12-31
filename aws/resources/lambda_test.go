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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockLambdaClient struct {
	DeleteFunctionOutput lambda.DeleteFunctionOutput
	ListFunctionsOutput  lambda.ListFunctionsOutput
	ListTagsOutput       lambda.ListTagsOutput
}

func (m *mockLambdaClient) DeleteFunction(ctx context.Context, params *lambda.DeleteFunctionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteFunctionOutput, error) {
	return &m.DeleteFunctionOutput, nil
}

func (m *mockLambdaClient) ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error) {
	return &m.ListFunctionsOutput, nil
}

func (m *mockLambdaClient) ListTags(ctx context.Context, params *lambda.ListTagsInput, optFns ...func(*lambda.Options)) (*lambda.ListTagsOutput, error) {
	return &m.ListTagsOutput, nil
}

func TestListLambdaFunctions(t *testing.T) {
	t.Parallel()

	testName1 := "test-lambda-function1"
	testName2 := "test-lambda-function2"

	layout := "2006-01-02T15:04:05.000+0000"
	testTimeStr := "2023-07-28T12:34:56.789+0000"
	testTime, err := time.Parse(layout, testTimeStr)
	require.NoError(t, err)

	mock := &mockLambdaClient{
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
			names, err := listLambdaFunctions(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteLambdaFunction(t *testing.T) {
	t.Parallel()

	mock := &mockLambdaClient{}
	err := deleteLambdaFunction(context.Background(), mock, aws.String("test"))
	require.NoError(t, err)
}

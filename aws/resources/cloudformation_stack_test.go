package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedCloudFormationStacks struct {
	CloudFormationStacksAPI
	ListStacksOutput      cloudformation.ListStacksOutput
	DescribeStacksOutputs map[string]cloudformation.DescribeStacksOutput
	DeleteStackOutput     cloudformation.DeleteStackOutput
}

func (m mockedCloudFormationStacks) ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
	return &m.ListStacksOutput, nil
}

func (m mockedCloudFormationStacks) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if output, exists := m.DescribeStacksOutputs[*params.StackName]; exists {
		return &output, nil
	}
	return &cloudformation.DescribeStacksOutput{}, nil
}

func (m mockedCloudFormationStacks) DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
	return &m.DeleteStackOutput, nil
}

func TestCloudFormationStackGetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-stack-1"
	testName2 := "test-stack-2"
	now := time.Now()

	mockClient := mockedCloudFormationStacks{
		ListStacksOutput: cloudformation.ListStacksOutput{
			StackSummaries: []types.StackSummary{
				{
					StackName:    aws.String(testName1),
					CreationTime: aws.Time(now),
					StackStatus:  types.StackStatusCreateComplete,
				},
				{
					StackName:    aws.String(testName2),
					CreationTime: aws.Time(now.Add(1)),
					StackStatus:  types.StackStatusUpdateComplete,
				},
			},
		},
		DescribeStacksOutputs: map[string]cloudformation.DescribeStacksOutput{
			testName1: {
				Stacks: []types.Stack{
					{
						StackName:    aws.String(testName1),
						CreationTime: aws.Time(now),
						Tags: []types.Tag{
							{Key: aws.String("Environment"), Value: aws.String("test")},
						},
					},
				},
			},
			testName2: {
				Stacks: []types.Stack{
					{
						StackName:    aws.String(testName2),
						CreationTime: aws.Time(now.Add(1)),
						Tags:         []types.Tag{},
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
					}},
				},
			},
			expected: []string{testName2},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
					},
				},
			},
			expected: []string{testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listCloudFormationStacks(
				context.Background(),
				mockClient,
				resource.Scope{Region: "us-east-1"},
				tc.configObj,
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestCloudFormationStackNuke(t *testing.T) {
	t.Parallel()

	mockClient := mockedCloudFormationStacks{
		DeleteStackOutput: cloudformation.DeleteStackOutput{},
	}

	err := deleteCloudFormationStack(
		context.Background(),
		mockClient,
		aws.String("test-stack"),
	)
	require.NoError(t, err)
}

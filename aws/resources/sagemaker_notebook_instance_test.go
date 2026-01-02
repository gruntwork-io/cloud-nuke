package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockSageMakerNotebookClient struct {
	ListNotebookInstancesOutput    sagemaker.ListNotebookInstancesOutput
	StopNotebookInstanceOutput     sagemaker.StopNotebookInstanceOutput
	DeleteNotebookInstanceOutput   sagemaker.DeleteNotebookInstanceOutput
	DescribeNotebookInstanceOutput sagemaker.DescribeNotebookInstanceOutput
}

func (m *mockSageMakerNotebookClient) ListNotebookInstances(ctx context.Context, params *sagemaker.ListNotebookInstancesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListNotebookInstancesOutput, error) {
	return &m.ListNotebookInstancesOutput, nil
}

func (m *mockSageMakerNotebookClient) StopNotebookInstance(ctx context.Context, params *sagemaker.StopNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.StopNotebookInstanceOutput, error) {
	return &m.StopNotebookInstanceOutput, nil
}

func (m *mockSageMakerNotebookClient) DeleteNotebookInstance(ctx context.Context, params *sagemaker.DeleteNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteNotebookInstanceOutput, error) {
	return &m.DeleteNotebookInstanceOutput, nil
}

func (m *mockSageMakerNotebookClient) DescribeNotebookInstance(ctx context.Context, params *sagemaker.DescribeNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeNotebookInstanceOutput, error) {
	return &m.DescribeNotebookInstanceOutput, nil
}

func TestListSageMakerNotebookInstances(t *testing.T) {
	t.Parallel()

	testName1 := "test-notebook-1"
	testName2 := "test-notebook-2"
	now := time.Now()

	mock := &mockSageMakerNotebookClient{
		ListNotebookInstancesOutput: sagemaker.ListNotebookInstancesOutput{
			NotebookInstances: []types.NotebookInstanceSummary{
				{NotebookInstanceName: aws.String(testName1), CreationTime: aws.Time(now)},
				{NotebookInstanceName: aws.String(testName2), CreationTime: aws.Time(now.Add(1 * time.Hour))},
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
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
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
			names, err := listSageMakerNotebookInstances(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestStopNotebookInstance(t *testing.T) {
	t.Parallel()

	mock := &mockSageMakerNotebookClient{}
	err := stopNotebookInstance(context.Background(), mock, aws.String("test-notebook"))
	require.NoError(t, err)
}

func TestDeleteNotebookInstance(t *testing.T) {
	t.Parallel()

	mock := &mockSageMakerNotebookClient{}
	err := deleteNotebookInstance(context.Background(), mock, aws.String("test-notebook"))
	require.NoError(t, err)
}

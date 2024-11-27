package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

// There's a built-in function WaitUntilDBInstanceAvailable but
// the times that it was tested, it wasn't returning anything so we'll leave with the
// custom one.

type mockedSageMakerNotebookInstance struct {
	SageMakerNotebookInstancesAPI
	ListNotebookInstancesOutput    sagemaker.ListNotebookInstancesOutput
	StopNotebookInstanceOutput     sagemaker.StopNotebookInstanceOutput
	DeleteNotebookInstanceOutput   sagemaker.DeleteNotebookInstanceOutput
	DescribeNotebookInstanceOutput sagemaker.DescribeNotebookInstanceOutput
	DescribeNotebookInstanceError  error
}

func (m mockedSageMakerNotebookInstance) ListNotebookInstances(ctx context.Context, params *sagemaker.ListNotebookInstancesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListNotebookInstancesOutput, error) {
	return &m.ListNotebookInstancesOutput, nil
}

func (m mockedSageMakerNotebookInstance) StopNotebookInstance(ctx context.Context, params *sagemaker.StopNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.StopNotebookInstanceOutput, error) {
	return &m.StopNotebookInstanceOutput, nil
}

func (m mockedSageMakerNotebookInstance) DeleteNotebookInstance(ctx context.Context, params *sagemaker.DeleteNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteNotebookInstanceOutput, error) {

	return &m.DeleteNotebookInstanceOutput, nil
}
func (m mockedSageMakerNotebookInstance) DescribeNotebookInstance(ctx context.Context, params *sagemaker.DescribeNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeNotebookInstanceOutput, error) {

	return &m.DescribeNotebookInstanceOutput, m.DescribeNotebookInstanceError
}

func (m mockedSageMakerNotebookInstance) WaitForOutput(ctx context.Context, params *sagemaker.DescribeNotebookInstanceInput, maxWaitDur time.Duration, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeNotebookInstanceOutput, error) {
	return nil, nil
}

func TestSageMakerNotebookInstances_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	testName1 := "test1"
	testName2 := "test2"
	smni := SageMakerNotebookInstances{
		Client: mockedSageMakerNotebookInstance{
			ListNotebookInstancesOutput: sagemaker.ListNotebookInstancesOutput{
				NotebookInstances: []types.NotebookInstanceSummary{
					{
						NotebookInstanceName: aws.String(testName1),
						CreationTime:         aws.Time(now),
					},
					{
						NotebookInstanceName: aws.String(testName2),
						CreationTime:         aws.Time(now.Add(1)),
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
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := smni.getAll(context.Background(), config.Config{
				SageMakerNotebook: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestSageMakerNotebookInstances_NukeAll(t *testing.T) {

	t.Parallel()

	testName1 := "test1"
	testName2 := "test2"
	now := time.Now()

	smni := SageMakerNotebookInstances{
		Client: mockedSageMakerNotebookInstance{
			ListNotebookInstancesOutput: sagemaker.ListNotebookInstancesOutput{
				NotebookInstances: []types.NotebookInstanceSummary{
					{
						NotebookInstanceName: aws.String(testName1),
						CreationTime:         aws.Time(now),
					},
					{
						NotebookInstanceName: aws.String(testName2),
						CreationTime:         aws.Time(now.Add(1)),
					},
				},
			},
			StopNotebookInstanceOutput:   sagemaker.StopNotebookInstanceOutput{},
			DeleteNotebookInstanceOutput: sagemaker.DeleteNotebookInstanceOutput{},
			DescribeNotebookInstanceOutput: sagemaker.DescribeNotebookInstanceOutput{
				NotebookInstanceStatus: types.NotebookInstanceStatusStopped,
				NotebookInstanceName:   aws.String(testName2),
			},
			DescribeNotebookInstanceError: &smithy.GenericAPIError{
				Code: "ValidationException",
			},
		},
	}

	smni.Context = context.Background()

	tests := []struct {
		name      string
		instances []*string
	}{
		{
			name:      "Single instance",
			instances: []*string{aws.String("test1")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := smni.nukeAll(tt.instances)
			require.NoError(t, err)
		})
	}
}

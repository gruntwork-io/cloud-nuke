package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/aws/aws-sdk-go/service/sagemaker/sagemakeriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

// There's a built-in function WaitUntilDBInstanceAvailable but
// the times that it was tested, it wasn't returning anything so we'll leave with the
// custom one.

type mockedSageMakerNotebookInstance struct {
	sagemakeriface.SageMakerAPI
	ListNotebookInstancesOutput  sagemaker.ListNotebookInstancesOutput
	StopNotebookInstanceOutput   sagemaker.StopNotebookInstanceOutput
	DeleteNotebookInstanceOutput sagemaker.DeleteNotebookInstanceOutput
}

func (m mockedSageMakerNotebookInstance) ListNotebookInstancesWithContext(_ awsgo.Context, _ *sagemaker.ListNotebookInstancesInput, _ ...request.Option) (*sagemaker.ListNotebookInstancesOutput, error) {
	return &m.ListNotebookInstancesOutput, nil
}

func (m mockedSageMakerNotebookInstance) StopNotebookInstanceWithContext(_ awsgo.Context, _ *sagemaker.StopNotebookInstanceInput, _ ...request.Option) (*sagemaker.StopNotebookInstanceOutput, error) {
	return &m.StopNotebookInstanceOutput, nil
}

func (m mockedSageMakerNotebookInstance) WaitUntilNotebookInstanceStoppedWithContext(_ awsgo.Context, _ *sagemaker.DescribeNotebookInstanceInput, _ ...request.WaiterOption) error {
	return nil
}

func (m mockedSageMakerNotebookInstance) WaitUntilNotebookInstanceDeletedWithContext(_ awsgo.Context, _ *sagemaker.DescribeNotebookInstanceInput, _ ...request.WaiterOption) error {
	return nil
}

func (m mockedSageMakerNotebookInstance) DeleteNotebookInstance(input *sagemaker.DeleteNotebookInstanceInput) (*sagemaker.DeleteNotebookInstanceOutput, error) {
	return &m.DeleteNotebookInstanceOutput, nil
}

func TestSageMakerNotebookInstances_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	testName1 := "test1"
	testName2 := "test2"
	smni := SageMakerNotebookInstances{
		Client: mockedSageMakerNotebookInstance{
			ListNotebookInstancesOutput: sagemaker.ListNotebookInstancesOutput{
				NotebookInstances: []*sagemaker.NotebookInstanceSummary{
					{
						NotebookInstanceName: awsgo.String(testName1),
						CreationTime:         awsgo.Time(now),
					},
					{
						NotebookInstanceName: awsgo.String(testName2),
						CreationTime:         awsgo.Time(now.Add(1)),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestSageMakerNotebookInstances_NukeAll(t *testing.T) {

	t.Parallel()

	smni := SageMakerNotebookInstances{
		Client: mockedSageMakerNotebookInstance{
			StopNotebookInstanceOutput:   sagemaker.StopNotebookInstanceOutput{},
			DeleteNotebookInstanceOutput: sagemaker.DeleteNotebookInstanceOutput{},
		},
	}

	err := smni.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}

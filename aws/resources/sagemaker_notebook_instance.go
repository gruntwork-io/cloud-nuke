package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

const (
	// notebookStopWaitDuration is the maximum time to wait for a notebook to stop.
	notebookStopWaitDuration = 10 * time.Minute
	// notebookDeleteWaitDuration is the maximum time to wait for a notebook to be deleted.
	notebookDeleteWaitDuration = 10 * time.Minute
)

// SageMakerNotebookInstancesAPI defines the interface for SageMaker Notebook Instance operations.
type SageMakerNotebookInstancesAPI interface {
	ListNotebookInstances(ctx context.Context, params *sagemaker.ListNotebookInstancesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListNotebookInstancesOutput, error)
	StopNotebookInstance(ctx context.Context, params *sagemaker.StopNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.StopNotebookInstanceOutput, error)
	DeleteNotebookInstance(ctx context.Context, params *sagemaker.DeleteNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteNotebookInstanceOutput, error)
	DescribeNotebookInstance(ctx context.Context, params *sagemaker.DescribeNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeNotebookInstanceOutput, error)
}

// NewSageMakerNotebookInstances creates a new SageMaker Notebook Instances resource
// using the generic resource pattern.
func NewSageMakerNotebookInstances() AwsResource {
	return NewAwsResource(&resource.Resource[SageMakerNotebookInstancesAPI]{
		ResourceTypeName: "sagemaker-notebook-smni",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SageMakerNotebookInstancesAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = sagemaker.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SageMakerNotebook
		},
		Lister: listSageMakerNotebookInstances,
		Nuker: resource.MultiStepDeleter(
			stopNotebookInstance,
			waitNotebookInstanceStopped,
			deleteNotebookInstance,
			waitNotebookInstanceDeleted,
		),
	})
}

// listSageMakerNotebookInstances retrieves all SageMaker notebook instances that match the config filters.
func listSageMakerNotebookInstances(ctx context.Context, client SageMakerNotebookInstancesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string

	paginator := sagemaker.NewListNotebookInstancesPaginator(client, &sagemaker.ListNotebookInstancesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, notebook := range page.NotebookInstances {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: notebook.NotebookInstanceName,
				Time: notebook.CreationTime,
			}) {
				names = append(names, notebook.NotebookInstanceName)
			}
		}
	}

	return names, nil
}

// stopNotebookInstance stops a SageMaker notebook instance.
// Stopping is required before deletion.
func stopNotebookInstance(ctx context.Context, client SageMakerNotebookInstancesAPI, name *string) error {
	_, err := client.StopNotebookInstance(ctx, &sagemaker.StopNotebookInstanceInput{
		NotebookInstanceName: name,
	})
	// Ignore error if the notebook is already stopped or stopping
	return err
}

// waitNotebookInstanceStopped waits for the notebook instance to reach the Stopped state.
func waitNotebookInstanceStopped(ctx context.Context, client SageMakerNotebookInstancesAPI, name *string) error {
	waiter := sagemaker.NewNotebookInstanceStoppedWaiter(client)
	return waiter.Wait(ctx, &sagemaker.DescribeNotebookInstanceInput{
		NotebookInstanceName: name,
	}, notebookStopWaitDuration)
}

// deleteNotebookInstance deletes a SageMaker notebook instance.
func deleteNotebookInstance(ctx context.Context, client SageMakerNotebookInstancesAPI, name *string) error {
	_, err := client.DeleteNotebookInstance(ctx, &sagemaker.DeleteNotebookInstanceInput{
		NotebookInstanceName: name,
	})
	return err
}

// waitNotebookInstanceDeleted waits for the notebook instance to be fully deleted.
func waitNotebookInstanceDeleted(ctx context.Context, client SageMakerNotebookInstancesAPI, name *string) error {
	waiter := sagemaker.NewNotebookInstanceDeletedWaiter(client)
	return waiter.Wait(ctx, &sagemaker.DescribeNotebookInstanceInput{
		NotebookInstanceName: name,
	}, notebookDeleteWaitDuration)
}

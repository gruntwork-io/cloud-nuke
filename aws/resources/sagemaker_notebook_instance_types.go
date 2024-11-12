package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

//	type SageMakerNotebookInstancesAPI interface {
//		ListNotebookInstances(ctx context.Context, params *sagemaker.ListNotebookInstancesInput, optFns ...func(*Options)) (*ListNotebookInstancesOutput, error)
//	}
type SageMakerNotebookInstancesAPI interface {
	DescribeNotebookInstance(ctx context.Context, params *sagemaker.DescribeNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DescribeNotebookInstanceOutput, error)
	ListNotebookInstances(ctx context.Context, params *sagemaker.ListNotebookInstancesInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListNotebookInstancesOutput, error)
	DeleteNotebookInstance(ctx context.Context, params *sagemaker.DeleteNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteNotebookInstanceOutput, error)
	StopNotebookInstance(ctx context.Context, params *sagemaker.StopNotebookInstanceInput, optFns ...func(*sagemaker.Options)) (*sagemaker.StopNotebookInstanceOutput, error)
}
type SageMakerNotebookInstances struct {
	BaseAwsResource
	Client        SageMakerNotebookInstancesAPI
	Region        string
	InstanceNames []string
}

func (smni *SageMakerNotebookInstances) InitV2(cfg aws.Config) {
	smni.Client = sagemaker.NewFromConfig(cfg)
}

func (smni *SageMakerNotebookInstances) IsUsingV2() bool { return true }

func (smni *SageMakerNotebookInstances) ResourceName() string {
	return "sagemaker-notebook-smni"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (smni *SageMakerNotebookInstances) ResourceIdentifiers() []string {
	return smni.InstanceNames
}

func (smni *SageMakerNotebookInstances) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (smni *SageMakerNotebookInstances) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SageMakerNotebook
}

func (smni *SageMakerNotebookInstances) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := smni.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	smni.InstanceNames = aws.ToStringSlice(identifiers)
	return smni.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (smni *SageMakerNotebookInstances) Nuke(identifiers []string) error {
	if err := smni.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type SageMakerNotebookInstanceDeleteError struct {
	name string
}

func (e SageMakerNotebookInstanceDeleteError) Error() string {
	return "SageMaker Notebook Instance:" + e.name + "was not deleted"
}

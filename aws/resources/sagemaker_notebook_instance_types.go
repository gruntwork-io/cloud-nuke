package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/aws/aws-sdk-go/service/sagemaker/sagemakeriface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SageMakerNotebookInstances struct {
	BaseAwsResource
	Client        sagemakeriface.SageMakerAPI
	Region        string
	InstanceNames []string
}

func (smni *SageMakerNotebookInstances) Init(session *session.Session) {
	smni.Client = sagemaker.New(session)
}

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

	smni.InstanceNames = awsgo.StringValueSlice(identifiers)
	return smni.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (smni *SageMakerNotebookInstances) Nuke(identifiers []string) error {
	if err := smni.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
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

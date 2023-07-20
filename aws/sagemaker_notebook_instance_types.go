package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sagemaker/sagemakeriface"
	"github.com/gruntwork-io/go-commons/errors"
)

type SageMakerNotebookInstance struct {
	Client        sagemakeriface.SageMakerAPI
	Region        string
	InstanceNames []string
}

func (instance SageMakerNotebookInstance) ResourceName() string {
	return "sagemaker-notebook-instance"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance SageMakerNotebookInstance) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance SageMakerNotebookInstance) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (instance SageMakerNotebookInstance) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllNotebookInstances(session, awsgo.StringSlice(identifiers)); err != nil {
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

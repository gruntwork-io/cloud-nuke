package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (smni *SageMakerNotebookInstances) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := smni.Client.ListNotebookInstancesWithContext(
		smni.Context,
		&sagemaker.ListNotebookInstancesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, notebook := range result.NotebookInstances {
		if configObj.SageMakerNotebook.ShouldInclude(config.ResourceValue{
			Name: notebook.NotebookInstanceName,
			Time: notebook.CreationTime,
		}) {
			names = append(names, notebook.NotebookInstanceName)
		}
	}

	return names, nil
}

func (smni *SageMakerNotebookInstances) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No Sagemaker Notebook Instance to nuke in region %s", smni.Region)
		return nil
	}

	logging.Debugf("Deleting all Sagemaker Notebook Instances in region %s", smni.Region)
	deletedNames := []*string{}

	for _, name := range names {
		_, err := smni.Client.StopNotebookInstanceWithContext(smni.Context, &sagemaker.StopNotebookInstanceInput{
			NotebookInstanceName: name,
		})
		if err != nil {
			logging.Errorf("[Failed] %s: %s", *name, err)
		}

		err = smni.Client.WaitUntilNotebookInstanceStoppedWithContext(smni.Context, &sagemaker.DescribeNotebookInstanceInput{
			NotebookInstanceName: name,
		})
		if err != nil {
			logging.Errorf("[Failed] %s", err)
		}

		_, err = smni.Client.DeleteNotebookInstance(&sagemaker.DeleteNotebookInstanceInput{
			NotebookInstanceName: name,
		})

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted Sagemaker Notebook Instance: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := smni.Client.WaitUntilNotebookInstanceDeletedWithContext(smni.Context, &sagemaker.DescribeNotebookInstanceInput{
				NotebookInstanceName: name,
			})

			// Record status of this resource
			e := report.Entry{
				Identifier:   aws.StringValue(name),
				ResourceType: "SageMaker Notebook Instance",
				Error:        err,
			}
			report.Record(e)

			if err != nil {
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Debugf("[OK] %d Sagemaker Notebook Instance(s) deleted in %s", len(deletedNames), smni.Region)
	return nil
}

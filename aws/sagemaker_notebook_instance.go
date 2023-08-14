package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (smni *SageMakerNotebookInstances) getAll(configObj config.Config) ([]*string, error) {
	result, err := smni.Client.ListNotebookInstances(&sagemaker.ListNotebookInstancesInput{})
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
		logging.Logger.Debugf("No Sagemaker Notebook Instance to nuke in region %s", smni.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Sagemaker Notebook Instances in region %s", smni.Region)
	deletedNames := []*string{}

	for _, name := range names {
		_, err := smni.Client.StopNotebookInstance(&sagemaker.StopNotebookInstanceInput{
			NotebookInstanceName: name,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *name, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Sagemaker Notebook Instance",
			}, map[string]interface{}{
				"region": smni.Region,
				"reason": "Failed to Stop Notebook",
			})
		}

		err = smni.Client.WaitUntilNotebookInstanceStopped(&sagemaker.DescribeNotebookInstanceInput{
			NotebookInstanceName: name,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Sagemaker Notebook Instance",
			}, map[string]interface{}{
				"region": smni.Region,
				"reason": "Failed waiting for notebook to stop",
			})
		}

		_, err = smni.Client.DeleteNotebookInstance(&sagemaker.DeleteNotebookInstanceInput{
			NotebookInstanceName: name,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *name, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Sagemaker Notebook Instance",
			}, map[string]interface{}{
				"region": smni.Region,
				"reason": "Failed to Delete Notebook",
			})
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted Sagemaker Notebook Instance: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := smni.Client.WaitUntilNotebookInstanceDeleted(&sagemaker.DescribeNotebookInstanceInput{
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
				logging.Logger.Errorf("[Failed] %s", err)
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking Sagemaker Notebook Instance",
				}, map[string]interface{}{
					"region": smni.Region,
					"reason": "Failed waiting for notebook instance to delete",
				})
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Logger.Debugf("[OK] %d Sagemaker Notebook Instance(s) deleted in %s", len(deletedNames), smni.Region)
	return nil
}

package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (smni *SageMakerNotebookInstances) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := smni.Client.ListNotebookInstances(
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
		_, err := smni.Client.StopNotebookInstance(smni.Context, &sagemaker.StopNotebookInstanceInput{
			NotebookInstanceName: name,
		})
		if err != nil {
			logging.Errorf("[Failed] %s: %s", *name, err)
		}

		err = WaitUntilNotebookInstanceStopped(smni.Context, smni.Client, name)
		if err != nil {
			logging.Errorf("[Failed] %s", err)
		}

		_, err = smni.Client.DeleteNotebookInstance(smni.Context, &sagemaker.DeleteNotebookInstanceInput{
			NotebookInstanceName: name,
		})

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted Sagemaker Notebook Instance: %s", aws.ToString(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {
			err := WaitUntilNotebookInstanceDeleted(smni.Context, smni.Client, name)

			// Record status of this resource
			e := report.Entry{
				Identifier:   aws.ToString(name),
				ResourceType: "SageMaker Notebook Instance",
				Error:        err,
			}
			go report.Record(e)

			if err != nil {
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Debugf("[OK] %d Sagemaker Notebook Instance(s) deleted in %s", len(deletedNames), smni.Region)
	return nil
}

func WaitUntilNotebookInstanceStopped(ctx context.Context, client SageMakerNotebookInstancesAPI, name *string) error {
	waiter := sagemaker.NewNotebookInstanceStoppedWaiter(client)

	for i := 0; i < maxStopRetries; i++ {
		logging.Debugf("Waiting for notebook instance (%s) to stop (attempt %d/%d)", *name, i+1, maxStopRetries)

		err := waiter.Wait(ctx, &sagemaker.DescribeNotebookInstanceInput{
			NotebookInstanceName: name,
		}, stopWaitDuration)

		if err == nil {
			logging.Debugf("Notebook instance (%s) has successfully stopped.", *name)
			return nil
		}

		logging.Debugf("Error during stop wait for notebook instance (%s): %v", *name, err)

		if i == maxStopRetries-1 {
			return fmt.Errorf("failed to confirm SageMaker notebook instance (%s) stop after %d attempts: %w", *name, maxStopRetries, err)
		}

		logging.Debugf("Retrying stop wait for notebook instance (%s) (attempt %d/%d)", *name, i+1, maxStopRetries)
		logging.Debugf("Underlying error was: %s", err)

	}

	return fmt.Errorf("unexpected error: reached end of retry loop for notebook instance stop (%s)", *name)
}

func WaitUntilNotebookInstanceDeleted(ctx context.Context, client SageMakerNotebookInstancesAPI, name *string) error {
	waiter := sagemaker.NewNotebookInstanceDeletedWaiter(client)

	for i := 0; i < maxRetries; i++ {
		logging.Debugf("Waiting until notebook instance (%s) deletion is propagated (attempt %d / %d)", *name, i+1, maxRetries)

		err := waiter.Wait(ctx, &sagemaker.DescribeNotebookInstanceInput{
			NotebookInstanceName: name,
		}, waitDuration)
		if err == nil {
			logging.Debugf("Successfully detected SageMaker notebook instance deletion.")
			return nil
		}

		logging.Debugf("Error during deletion wait for notebook instance (%s): %v", *name, err)

		if i == maxRetries-1 {
			return fmt.Errorf("failed to confirm deletion of SageMaker notebook instance (%s) after %d attempts: %w", *name, maxRetries, err)
		}

		logging.Debugf("Retrying deletion wait for notebook instance (%s) (attempt %d / %d)", *name, i+1, maxRetries)
		logging.Debugf("Underlying error was: %s", err)

	}

	return fmt.Errorf("unexpected error: reached end of retry loop for notebook instance deletion (%s)", *name)
}

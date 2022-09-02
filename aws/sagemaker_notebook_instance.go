package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllNotebookInstances(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := sagemaker.New(session)

	result, err := svc.ListNotebookInstances(&sagemaker.ListNotebookInstancesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, notebook := range result.NotebookInstances {
		if notebook.CreationTime == nil {
			continue
		}
		if !excludeAfter.After(awsgo.TimeValue(notebook.CreationTime)) {
			continue
		}
		if !config.ShouldInclude(awsgo.StringValue(notebook.NotebookInstanceName), configObj.S3.IncludeRule.NamesRegExp, configObj.S3.ExcludeRule.NamesRegExp) {
			continue
		}
		names = append(names, notebook.NotebookInstanceName)
	}

	return names, nil
}

func nukeAllNotebookInstances(session *session.Session, names []*string) error {
	svc := sagemaker.New(session)

	if len(names) == 0 {
		logging.Logger.Infof("No Sagemaker Notebook Instance to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all Sagemaker Notebook Instances in region %s", *session.Config.Region)
	deletedNames := []*string{}

	for _, name := range names {
		params := &sagemaker.DeleteNotebookInstanceInput{
			NotebookInstanceName: name,
		}

		_, err := svc.StopNotebookInstance(&sagemaker.StopNotebookInstanceInput{
			NotebookInstanceName: name,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *name, err)
		}

		err = svc.WaitUntilNotebookInstanceStopped(&sagemaker.DescribeNotebookInstanceInput{
			NotebookInstanceName: name,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		_, err = svc.DeleteNotebookInstance(params)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Infof("Deleted Sagemaker Notebook Instance: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := svc.WaitUntilNotebookInstanceDeleted(&sagemaker.DescribeNotebookInstanceInput{
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
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Logger.Infof("[OK] %d Sagemaker Notebook Instance(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}

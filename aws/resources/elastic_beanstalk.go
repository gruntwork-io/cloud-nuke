package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeEBApplication(app *elasticbeanstalk.ApplicationDescription, configObj config.Config) bool {
	return configObj.ElasticBeanstalk.ShouldInclude(config.ResourceValue{
		Name: app.ApplicationName,
		Time: app.DateCreated,
	})
}

// Returns a formatted string of EB application ids
func (eb *EBApplications) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	output, err := eb.Client.DescribeApplicationsWithContext(eb.Context, &elasticbeanstalk.DescribeApplicationsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	var appIds []*string
	for _, app := range output.Applications {
		if shouldIncludeEBApplication(app, configObj) {
			appIds = append(appIds, app.ApplicationName)
		}
	}
	return appIds, nil
}

// Deletes all EB Applications
func (eb *EBApplications) nukeAll(appIds []*string) error {
	if len(appIds) == 0 {
		logging.Debugf("No Elastic Beanstalk to nuke in region %s", eb.Region)
		return nil
	}

	logging.Debugf("Deleting all Elastic Beanstalk applications in region %s", eb.Region)
	var deletedApps []*string

	for _, id := range appIds {
		_, err := eb.Client.DeleteApplicationWithContext(eb.Context, &elasticbeanstalk.DeleteApplicationInput{
			ApplicationName:     id,
			TerminateEnvByForce: aws.Bool(true),
		})
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "Elastic Beanstalk Application",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
			continue
		}

		// get the deleted ids
		deletedApps = append(deletedApps, id)
		logging.Debugf("Deleted Elastic Beanstalk application: %s", *id)

	}
	logging.Debugf("[OK] %d Elastic Beanstalk application(s) deleted in %s", len(deletedApps), eb.Region)
	return nil
}

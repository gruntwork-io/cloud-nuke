package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func getAllElasticBeanstalkApplications(sess *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
	svc := elasticbeanstalk.New(sess)

	applicationNames := []string{}
	describeApplicationsInput := &elasticbeanstalk.DescribeApplicationsInput{}

	output, err := svc.DescribeApplications(describeApplicationsInput)

	for _, application := range output.Applications {
		if shouldIncludeElasticBeanstalkApplication(application, excludeAfter, configObj) {
			applicationNames = append(applicationNames, aws.StringValue(application.ApplicationName))
		}
	}

	if err != nil {
		return []string{}, err
	}

	return applicationNames, nil
}

func shouldIncludeElasticBeanstalkApplication(application *elasticbeanstalk.ApplicationDescription, excludeAfter time.Time, configObj config.Config) bool {
	return config.ShouldInclude(
		aws.StringValue(application.ApplicationName),
		configObj.ElasticBeanstalkApplication.IncludeRule.NamesRegExp,
		configObj.ElasticBeanstalkApplication.ExcludeRule.NamesRegExp,
	)
}

func nukeAllElasticBeanstalkApplications(session *session.Session, names []*string) error {
	svc := elasticbeanstalk.New(session)

	if len(names) == 0 {
		logging.Logger.Debugf("No Elastic Beanstalk Applications to nuke in region %s", *session.Config.Region)

		return nil
	}

	var deletedNames []*string
	for _, name := range names {
		params := &elasticbeanstalk.DeleteApplicationInput{
			ApplicationName:     name,
			TerminateEnvByForce: aws.Bool(true),
		}
		_, err := svc.DeleteApplication(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(name),
			ResourceType: "ElasticBeanstalk Application",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted Elastic Beanstalk Application: %s", aws.StringValue(name))
		}
	}

	logging.Logger.Debugf("[OK] %d Elastic Beanstalk Applications(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}

package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/hashicorp/go-multierror"
)

func getAllElasticBeanstalkEnvironments(sess *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
	svc := elasticbeanstalk.New(sess)

	environments := []string{}
	describeEnvironmentsInput := &elasticbeanstalk.DescribeEnvironmentsInput{}

	output, err := svc.DescribeEnvironments(describeEnvironmentsInput)

	for _, environment := range output.Environments {
		if shouldIncludeElasticBeanstalkEnvironment(environment, excludeAfter, configObj) {
			environments = append(environments, aws.StringValue(environment.EnvironmentName))
		}
	}

	if err != nil {
		return []string{}, err
	}

	return environments, nil
}

func shouldIncludeElasticBeanstalkEnvironment(environment *elasticbeanstalk.EnvironmentDescription, excludeAfter time.Time, configObj config.Config) bool {
	return config.ShouldInclude(
		*environment.EnvironmentName,
		configObj.ElasticBeanstalkEnvironment.IncludeRule.NamesRegExp,
		configObj.ElasticBeanstalkEnvironment.ExcludeRule.NamesRegExp,
	)
}

func nukeAllElasticBeanstalkEnvironments(session *session.Session, environmentNamesonmentNames []*string) error {
	svc := elasticbeanstalk.New(session)

	if len(environmentNamesonmentNames) == 0 {
		logging.Logger.Debugf("No Elastic Beanstalk Environments to nuke in region %s", *session.Config.Region)

		return nil
	}

	var allErrs *multierror.Error

	var deletedNames []*string
	for _, environmentName := range environmentNamesonmentNames {
		params := &elasticbeanstalk.TerminateEnvironmentInput{
			EnvironmentName:    environmentName,
			TerminateResources: aws.Bool(true),
			ForceTerminate:     aws.Bool(true),
		}
		_, err := svc.TerminateEnvironment(params)
		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		}

		// Wait on deletion of the environment. This is slow, but it will reduce the likelihood of
		// propagation delays causing the environment to be deleted but the resources to still exist (leading to test failure)
		waitErr := svc.WaitUntilEnvironmentTerminated(&elasticbeanstalk.DescribeEnvironmentsInput{
			EnvironmentNames: []*string{environmentName},
		})

		if waitErr != nil {
			allErrs = multierror.Append(allErrs, waitErr)
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   *environmentName,
			ResourceType: "ElasticBeanstalk Environment",
			Error:        allErrs.ErrorOrNil(),
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, environmentName)
			logging.Logger.Debugf("Deleted Elastic Beanstalk Environment: %s", *environmentName)
		}
	}

	logging.Logger.Debugf("[OK] %d Elastic Beanstalk Environment(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}

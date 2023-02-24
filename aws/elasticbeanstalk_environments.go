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

func nukeAllElasticBeanstalkEnvironments(session *session.Session, names []*string) error {
	svc := elasticbeanstalk.New(session)

	if len(names) == 0 {
		logging.Logger.Debugf("No Elastic Beanstalk Environments to nuke in region %s", *session.Config.Region)

		return nil
	}

	var deletedNames []*string
	for _, name := range names {
		params := &elasticbeanstalk.TerminateEnvironmentInput{
			EnvironmentName:    name,
			TerminateResources: aws.Bool(true),
			ForceTerminate:     aws.Bool(true),
		}
		_, err := svc.TerminateEnvironment(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   *name,
			ResourceType: "ElasticBeanstalk Environment",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted Elastic Beanstalk Environment: %s", *name)
		}
	}

	logging.Logger.Debugf("[OK] %d Elastic Beanstalk Environment(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}

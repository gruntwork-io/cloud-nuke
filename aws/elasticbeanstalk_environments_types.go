package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type ElasticBeanstalkEnvironments struct {
	EnvironmentArns []string
}

func (e ElasticBeanstalkEnvironments) ResourceName() string {
	return "elasticbeanstalk-environments"
}

func (e ElasticBeanstalkEnvironments) ResourceIdentifiers() []string {
	return e.EnvironmentArns
}

func (e ElasticBeanstalkEnvironments) MaxBatchSize() int {
	return 100
}

func (e ElasticBeanstalkEnvironments) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElasticBeanstalkEnvironments(session, aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

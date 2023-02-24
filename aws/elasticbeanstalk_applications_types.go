package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type ElasticBeanstalkApplications struct {
	ApplicationArns []string
}

func (e ElasticBeanstalkApplications) ResourceName() string {
	return "elasticbeanstalk-applications"
}

func (e ElasticBeanstalkApplications) ResourceIdentifiers() []string {
	return e.ApplicationArns
}

func (e ElasticBeanstalkApplications) MaxBatchSize() int {
	return 100
}

func (e ElasticBeanstalkApplications) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElasticBeanstalkApplications(session, aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

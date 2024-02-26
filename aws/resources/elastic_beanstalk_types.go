package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk/elasticbeanstalkiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// EBSVolumes - represents all ebs volumes
type EBApplications struct {
	BaseAwsResource
	Client elasticbeanstalkiface.ElasticBeanstalkAPI
	Region string
	appIds []string
}

func (eb *EBApplications) Init(session *session.Session) {
	eb.Client = elasticbeanstalk.New(session)
}

// ResourceName - the simple name of the aws resource
func (eb *EBApplications) ResourceName() string {
	return "elastic-beanstalk"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (eb *EBApplications) ResourceIdentifiers() []string {
	return eb.appIds
}

func (eb *EBApplications) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (eb *EBApplications) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := eb.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	eb.appIds = awsgo.StringValueSlice(identifiers)
	return eb.appIds, nil
}

// Nuke - nuke 'em all!!!
func (eb *EBApplications) Nuke(identifiers []string) error {
	if err := eb.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

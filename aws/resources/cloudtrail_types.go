package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/cloudtrail/cloudtrailiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchLogGroup - represents all ec2 instances
type CloudtrailTrail struct {
	Client cloudtrailiface.CloudTrailAPI
	Region string
	Arns   []string
}

func (ct *CloudtrailTrail) Init(session *session.Session) {
	ct.Client = cloudtrail.New(session)
}

// ResourceName - the simple name of the aws resource
func (ct *CloudtrailTrail) ResourceName() string {
	return "cloudtrail"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (ct *CloudtrailTrail) ResourceIdentifiers() []string {
	return ct.Arns
}

func (ct *CloudtrailTrail) MaxBatchSize() int {
	return 50
}

func (ct *CloudtrailTrail) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ct.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ct.Arns = awsgo.StringValueSlice(identifiers)
	return ct.Arns, nil
}

// Nuke - nuke 'em all!!!
func (ct *CloudtrailTrail) Nuke(identifiers []string) error {
	if err := ct.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

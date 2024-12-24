package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudtrailTrailAPI interface {
	ListTrails(ctx context.Context, params *cloudtrail.ListTrailsInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.ListTrailsOutput, error)
	DeleteTrail(ctx context.Context, params *cloudtrail.DeleteTrailInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.DeleteTrailOutput, error)
}

// CloudtrailTrail - represents all CloudTrails
type CloudtrailTrail struct {
	BaseAwsResource
	Client CloudtrailTrailAPI
	Region string
	Arns   []string
}

func (ct *CloudtrailTrail) Init(cfg aws.Config) {
	ct.Client = cloudtrail.NewFromConfig(cfg)
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

func (ct *CloudtrailTrail) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudtrailTrail
}

func (ct *CloudtrailTrail) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ct.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ct.Arns = aws.ToStringSlice(identifiers)
	return ct.Arns, nil
}

// Nuke - nuke 'em all!!!
func (ct *CloudtrailTrail) Nuke(identifiers []string) error {
	if err := ct.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CodeDeployApplicationsAPI interface {
	ListApplications(ctx context.Context, params *codedeploy.ListApplicationsInput, optFns ...func(*codedeploy.Options)) (*codedeploy.ListApplicationsOutput, error)
	BatchGetApplications(ctx context.Context, params *codedeploy.BatchGetApplicationsInput, optFns ...func(*codedeploy.Options)) (*codedeploy.BatchGetApplicationsOutput, error)
	DeleteApplication(ctx context.Context, params *codedeploy.DeleteApplicationInput, optFns ...func(*codedeploy.Options)) (*codedeploy.DeleteApplicationOutput, error)
}

// CodeDeployApplications - represents all codedeploy applications
type CodeDeployApplications struct {
	BaseAwsResource
	Client   CodeDeployApplicationsAPI
	Region   string
	AppNames []string
}

func (cda *CodeDeployApplications) InitV2(cfg aws.Config) {
	cda.Client = codedeploy.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (cda *CodeDeployApplications) ResourceName() string {
	return "codedeploy-application"
}

// ResourceIdentifiers - The instance ids of the code deploy applications
func (cda *CodeDeployApplications) ResourceIdentifiers() []string {
	return cda.AppNames
}

func (cda *CodeDeployApplications) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle.
	return 100
}

func (cda *CodeDeployApplications) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CodeDeployApplications
}

func (cda *CodeDeployApplications) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cda.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cda.AppNames = aws.ToStringSlice(identifiers)
	return cda.AppNames, nil
}

// Nuke - nuke 'em all!!!
func (cda *CodeDeployApplications) Nuke(identifiers []string) error {
	if err := cda.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

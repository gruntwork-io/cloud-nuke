package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/aws/aws-sdk-go/service/codedeploy/codedeployiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// CodeDeployApplications - represents all codedeploy applications
type CodeDeployApplications struct {
	BaseAwsResource
	Client   codedeployiface.CodeDeployAPI
	Region   string
	AppNames []string
}

func (cda *CodeDeployApplications) Init(session *session.Session) {
	cda.Client = codedeploy.New(session)
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

func (cda *CodeDeployApplications) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cda.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cda.AppNames = awsgo.StringValueSlice(identifiers)
	return cda.AppNames, nil
}

// Nuke - nuke 'em all!!!
func (cda *CodeDeployApplications) Nuke(identifiers []string) error {
	if err := cda.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

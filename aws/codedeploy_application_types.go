package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codedeploy/codedeployiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// CodeDeployApplication - represents all codedeploy applications
type CodeDeployApplication struct {
	Client   codedeployiface.CodeDeployAPI
	Region   string
	AppNames []string
}

// ResourceName - the simple name of the aws resource
func (c CodeDeployApplication) ResourceName() string {
	return "codedeploy-application"
}

// ResourceIdentifiers - The instance ids of the code deploy applications
func (c CodeDeployApplication) ResourceIdentifiers() []string {
	return c.AppNames
}

func (c CodeDeployApplication) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle.
	return 100
}

// Nuke - nuke 'em all!!!
func (c CodeDeployApplication) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllCodeDeployApplications(session, identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

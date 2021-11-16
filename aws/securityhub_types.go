package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type SecurityHub struct {
	Insights []string
}

// ResourceName - the simple name of the aws resource
func (u SecurityHub) ResourceName() string {
	return "securityhub"
}

// ResourceIdentifiers -
func (u SecurityHub) ResourceIdentifiers() []string {
	return u.Insights
}

// Tentative batch size to ensure AWS doesn't throttle
func (u SecurityHub) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (u SecurityHub) Nuke(session *session.Session, insights []string) error {
	if err := nukeAllSecurityHubInsights(session, awsgo.StringSlice(insights)); err != nil {
		return errors.WithStackTrace(err)
	}

	if err := disableSecurityHub(session); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

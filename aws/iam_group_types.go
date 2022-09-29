package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

//IAMGroups - represents all IAMGroups on the AWS Account
type IAMGroups struct {
	GroupNames []string
}

//ResourceName - the simple name of the AWS resource
func (u IAMGroups) ResourceName() string {
	return "iam-group"
}

// ResourceIdentifiers - The IAM GroupNames
func (g IAMGroups) ResourceIdentifiers() []string {
	return g.GroupNames
}

// Tentative batch size to ensure AWS doesn't throttle
// There's a global max of 500 groups so it shouldn't take long either way
func (g IAMGroups) MaxBatchSize() int {
	return 80
}

// Nuke - Destroy every group in this collection
func (g IAMGroups) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllIamGroups(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

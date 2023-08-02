package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMUsers - represents all IAMUsers on the AWS Account
type IAMUsers struct {
	Client    iamiface.IAMAPI
	UserNames []string
}

// ResourceName - the simple name of the aws resource
func (iu IAMUsers) ResourceName() string {
	return "iam"
}

// ResourceIdentifiers - The IAM UserNames
func (iu IAMUsers) ResourceIdentifiers() []string {
	return iu.UserNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (iu IAMUsers) MaxBatchSize() int {
	return 49
}

// Nuke - nuke 'em all!!!
func (iu IAMUsers) Nuke(session *session.Session, users []string) error {
	if err := iu.nukeAll(awsgo.StringSlice(users)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

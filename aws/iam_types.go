package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// IAM - represents all users
type IamUsers struct {
	userNames []string
}

// ResourceName - the simple name of the aws resource
func (userName IamUsers) ResourceName() string {
	return "iam"
}

// ResourceIdentifiers - The IAM usernames
func (userName IamUsers) ResourceIdentifiers() []string {
	return userName.userNames
}

func (userName IamUsers) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

// Nuke - nuke 'em all!!!
func (userName IamUsers) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllIamUsers(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type UserNameAvailableError struct{}

func (e UserNameAvailableError) Error() string {
	return "Username is not available"
}

package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a slice of IAM UserNames
// TODO: Implement exclusion by time filter
func getAllIamUsers(session *session.Session, region string) ([]*string, error) {
	svc := iam.New(session)
	input := &iam.ListUsersInput{}

	var userNames []*string

	// TODO: Probably use ListUsers together with ListUsersPages in case there are lots of users
	output, err := svc.ListUsers(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, user := range output.Users {
		userNames = append(userNames, user.UserName)
	}

	return userNames, nil
}

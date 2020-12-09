package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// List all IAM users in the AWS account and returns a slice of the UserNames
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

// Delete all IAM Users
func nukeAllIamUsers(session *session.Session, userNames []*string) error {
	if len(userNames) == 0 {
		logging.Logger.Info("No IAM Users to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all IAM Users")

	deletedUsers := 0
	svc := iam.New(session)

	for _, userName := range userNames {
		input := &iam.DeleteUserInput{
			UserName: userName,
		}

		_, err := svc.DeleteUser(input)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedUsers++
			logging.Logger.Infof("Deleted IAM User: %s", *userName)
		}
	}

	logging.Logger.Infof("[OK] %d IAM User(s) terminated", deletedUsers)
	return nil
}
package aws

import (
	"regexp"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func includeUserByREList(userName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(userName) {
			return true
		}
	}
	return false
}

func excludeUserByREList(userName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(userName) {
			return false
		}
	}

	return true
}

func shouldIncludeUser(userName string, includeNamesREList []*regexp.Regexp, excludeNamesREList []*regexp.Regexp) bool {
	shouldInclude := false

	if len(includeNamesREList) > 0 {
		// If any include rules are specified,
		// only check to see if an exclude rule matches when an include rule matches the user
		if includeUserByREList(userName, includeNamesREList) {
			shouldInclude = excludeUserByREList(userName, excludeNamesREList)
		}
	} else if len(excludeNamesREList) > 0 {
		// Only check to see if an exclude rule matches when there are no include rules defined
		shouldInclude = excludeUserByREList(userName, excludeNamesREList)
	} else {
		shouldInclude = true
	}

	return shouldInclude
}

// List all IAM users in the AWS account and returns a slice of the UserNames
// TODO: Implement exclusion by time filter
// TODO: AWS IAM is global, specifying a region doesn't make sense and creates duplicated output
func getAllIamUsers(session *session.Session, region string, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)
	input := &iam.ListUsersInput{}

	var userNames []*string

	// TODO: Probably use ListUsers together with ListUsersPages in case there are lots of users
	output, err := svc.ListUsers(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, user := range output.Users {
		if shouldIncludeUser(*user.UserName, configObj.IAMUsers.IncludeRule.NamesRE, configObj.IAMUsers.ExcludeRule.NamesRE) {
			userNames = append(userNames, user.UserName)
		}
	}

	return userNames, nil
}

// TODO: This is only deleting the user but no associated resources
// According to https://docs.aws.amazon.com/sdk-for-go/api/service/iam/#IAM.DeleteUser
// "you must delete the items attached to the user manually, or the deletion fails"

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

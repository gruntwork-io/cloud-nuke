package aws

import (
	"github.com/gruntwork-io/cloud-nuke/logging"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that we can list IAM groups in an AWS account
func TestListIamGroups(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	localSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	groupNames, err := getAllIamGroups(localSession, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotEmpty(t, groupNames)
}

// Creates an empty IAM group for testing
func createEmptyTestGroup(t *testing.T, session *session.Session, name string) error {
	svc := iam.New(session)

	groupInput := &iam.CreateGroupInput{
		GroupName: awsgo.String(name),
	}

	_, err := svc.CreateGroup(groupInput)
	require.NoError(t, err)
	return nil
}

// Stores information for cleanup
type groupInfo struct {
	GroupName *string
	UserName  *string
	PolicyArn *string
}

func createNonEmptyTestGroup(t *testing.T, session *session.Session, groupName string, userName string) (*groupInfo, error) {
	svc := iam.New(session)

	//Create User
	userInput := &iam.CreateUserInput{
		UserName: awsgo.String(userName),
	}

	_, err := svc.CreateUser(userInput)
	require.NoError(t, err)

	//Create Policy
	policyOutput, err := svc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyDocument: awsgo.String(`{
			"Version": "2012-10-17",
			"Statement": [
					{
							"Sid": "VisualEditor0",
							"Effect": "Allow",
							"Action": "ec2:DescribeInstances",
							"Resource": "*"
					}
			]
		}`),
		PolicyName:  awsgo.String("policy-" + groupName),
		Description: awsgo.String("Policy created by cloud-nuke tests - Should be deleted"),
	})
	require.NoError(t, err)

	//Create Group
	groupInput := &iam.CreateGroupInput{
		GroupName: awsgo.String(groupName),
	}

	_, err = svc.CreateGroup(groupInput)
	require.NoError(t, err)

	//Add user to Group
	userGroupLinkInput := &iam.AddUserToGroupInput{
		GroupName: awsgo.String(groupName),
		UserName:  awsgo.String(userName),
	}
	_, err = svc.AddUserToGroup(userGroupLinkInput)
	require.NoError(t, err)

	//Add policy to Group

	groupPolicyInput := &iam.AttachGroupPolicyInput{
		PolicyArn: policyOutput.Policy.Arn,
		GroupName: awsgo.String(groupName),
	}
	_, err = svc.AttachGroupPolicy(groupPolicyInput)

	info := &groupInfo{
		GroupName: &groupName,
		PolicyArn: policyOutput.Policy.Arn,
		UserName:  &userName,
	}

	return info, nil
}

func deleteGroupExtraResources(session *session.Session, info *groupInfo) {
	svc := iam.New(session)
	_, err := svc.DeletePolicy(&iam.DeletePolicyInput{
		PolicyArn: info.PolicyArn,
	})
	if err != nil {
		logging.Logger.Errorf("Unable to delete test policy: %s", *info.PolicyArn)
	}

	_, err = svc.DeleteUser(&iam.DeleteUserInput{
		UserName: info.UserName,
	})
	if err != nil {
		logging.Logger.Errorf("Unable to delete test user: %s", *info.UserName)
	}
}

// Test that we can nuke iam groups.
func TestNukeIamGroups(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	localSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	//Create test entities
	emptyName := "cloud-nuke-test" + util.UniqueID()
	err = createEmptyTestGroup(t, localSession, emptyName)
	require.NoError(t, err)

	nonEmptyName := "cloud-nuke-test" + util.UniqueID()
	userName := "cloud-nuke-test" + util.UniqueID()
	info, err := createNonEmptyTestGroup(t, localSession, nonEmptyName, userName)
	defer deleteGroupExtraResources(localSession, info)
	require.NoError(t, err)

	//Assert test entities exist
	groupNames, err := getAllIamGroups(localSession, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(groupNames), nonEmptyName)
	assert.Contains(t, awsgo.StringValueSlice(groupNames), emptyName)

	//Nuke test entities
	err = nukeAllIamGroups(localSession, []*string{&emptyName, &nonEmptyName})
	require.NoError(t, err)

	//Assert test entities don't exist anymore
	groupNames, err = getAllIamGroups(localSession, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(groupNames), nonEmptyName)
	assert.NotContains(t, awsgo.StringValueSlice(groupNames), emptyName)
}

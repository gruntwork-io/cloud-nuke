package aws

import (
	"fmt"
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

//Test that we can list IAM groups in an AWS account
func TestListIamGroups(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	groupNames, err := getAllIamGroups(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotEmpty(t, groupNames)
}

//Creates an empty IAM group for testing
func createEmptyTestGroup(t *testing.T, session *session.Session, name string) error {
	svc := iam.New(session)

	groupInput := &iam.CreateGroupInput{
		GroupName: awsgo.String(name),
	}

	group, err := svc.CreateGroup(groupInput)
	fmt.Println(group.Group.Arn)
	require.NoError(t, err)
	return nil
}

func createNonEmptyTestGroup(t *testing.T, session *session.Session, groupName string, userName string) error {
	svc := iam.New(session)

	//Create User
	userInput := &iam.CreateUserInput{
		UserName: awsgo.String(userName),
	}

	_, err := svc.CreateUser(userInput)
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

	return nil
}

//Test that we can nuke iam groups.
func TestNukeIamGroups(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	//Create test entities
	emptyName := "cloud-nuke-test" + util.UniqueID()
	err = createEmptyTestGroup(t, session, emptyName)
	require.NoError(t, err)

	nonEmptyName := "cloud-nuke-test" + util.UniqueID()
	userName := "cloud-nuke-test" + util.UniqueID()
	err = createNonEmptyTestGroup(t, session, nonEmptyName, userName)
	require.NoError(t, err)

	//Assert test entities exist
	groupNames, err := getAllIamGroups(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(groupNames), nonEmptyName)
	assert.Contains(t, awsgo.StringValueSlice(groupNames), emptyName)

	//Nuke test entities
	err = nukeAllIamGroups(session, []*string{&emptyName, &nonEmptyName})
	require.NoError(t, err)

	err = nukeAllIamUsers(session, []*string{&userName})
	require.NoError(t, err)

	//Assert test entites don't exist anymore
	groupNames, err = getAllIamGroups(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(groupNames), nonEmptyName)
	assert.NotContains(t, awsgo.StringValueSlice(groupNames), emptyName)
}

//TODO could test filtered nuke if time

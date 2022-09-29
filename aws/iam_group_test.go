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

//TODO could create empty and full IAM groups?

//Creates an empty IAM group for testing
func createTestGroup(t *testing.T, session *session.Session, name string) error {
	svc := iam.New(session)

	groupInput := &iam.CreateGroupInput{
		GroupName: awsgo.String(name),
	}

	group, err := svc.CreateGroup(groupInput)
	fmt.Println(group.Group.Arn)
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

	name := "cloud-nuke-test" + util.UniqueID()
	err = createTestGroup(t, session, name)
	require.NoError(t, err)

	err = nukeAllIamGroups(session, []*string{&name})
	require.NoError(t, err)
}

package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIamUsers(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	// TODO: Implement exclusion by time filter
	// userNames, err := getAllIamUsers(session, region, time.Now().Add(1*time.Hour*-1))
	userNames, err := getAllIamUsers(session, region)
	require.NoError(t, err)

	// TODO: Remove this, just for temporary visual confirmation
	for _, name := range userNames {
		fmt.Printf("this is the name: %s\n", awsgo.StringValue(name))
	}

	assert.NotEmpty(t, userNames)
}

func createTestUser(t *testing.T, session *session.Session, name string) error {
	svc := iam.New(session)
	input := &iam.CreateUserInput{
		UserName: aws.String(name),
	}

	_, err := svc.CreateUser(input)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func TestCreateIamUser(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	name := "cloud-nuke-test-" + util.UniqueID()
	userNames, err := getAllIamUsers(session, region)
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(userNames), name)

	err = createTestUser(t, session, name)
	require.NoError(t, err)

	userNames, err = getAllIamUsers(session, region)
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(userNames), name)
}

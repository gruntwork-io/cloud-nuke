package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
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

	userNames, err := getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotEmpty(t, userNames)
}

func createTestUser(t *testing.T, session *session.Session, name string) error {
	svc := iam.New(session)
	input := &iam.CreateUserInput{
		UserName: aws.String(name),
	}

	_, err := svc.CreateUser(input)
	require.NoError(t, err)

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
	userNames, err := getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(userNames), name)

	err = createTestUser(t, session, name)
	defer nukeAllIamUsers(session, []*string{&name})
	require.NoError(t, err)

	userNames, err = getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(userNames), name)
}

func TestNukeIamUsers(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	name := "cloud-nuke-test-" + util.UniqueID()
	err = createTestUser(t, session, name)
	require.NoError(t, err)

	err = nukeAllIamUsers(session, []*string{&name})
	require.NoError(t, err)
}

func TestTimeFilterExclusionNewlyCreatedIamUser(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	// Assert user didn't exist
	name := "cloud-nuke-test-" + util.UniqueID()
	userNames, err := getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(userNames), name)

	// Creates a user
	err = createTestUser(t, session, name)
	defer nukeAllIamUsers(session, []*string{&name})

	// Assert user is created
	userNames, err = getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(userNames), name)

	// Assert user doesn't appear when we look at users older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	userNames, err = getAllIamUsers(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(userNames), name)
}

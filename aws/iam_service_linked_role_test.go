package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
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

func TestListIamServiceLinkedRoles(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	roleNames, err := getAllIamServiceLinkedRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotEmpty(t, roleNames)
}

func createTestServiceLinkedRole(t *testing.T, session *session.Session, name, awsServiceName string) error {
	svc := iam.New(session)

	input := &iam.CreateServiceLinkedRoleInput{
		AWSServiceName: aws.String(awsServiceName),
		Description:    aws.String("cloud-nuke-test"),
		CustomSuffix:   aws.String(name),
	}

	_, err := svc.CreateServiceLinkedRole(input)
	require.NoError(t, err)

	return nil
}

func TestCreateIamServiceLinkedRole(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	name := "cloud-nuke-test-" + util.UniqueID()
	awsServiceName := "autoscaling.amazonaws.com"
	iamServiceLinkedRoleName := "AWSServiceRoleForAutoScaling_" + name
	roleNames, err := getAllIamServiceLinkedRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(roleNames), name)

	err = createTestServiceLinkedRole(t, session, name, awsServiceName)
	require.NoError(t, err)

	roleNames, err = getAllIamServiceLinkedRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)
	//AWSServiceRoleForAutoScaling_cloud-nuke-test
	assert.Contains(t, awsgo.StringValueSlice(roleNames), iamServiceLinkedRoleName)
}

func TestNukeIamServiceLinkedRoles(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	name := "cloud-nuke-test-" + util.UniqueID()
	awsServiceName := "autoscaling.amazonaws.com"
	iamServiceLinkedRoleName := "AWSServiceRoleForAutoScaling_" + name

	err = createTestServiceLinkedRole(t, session, name, awsServiceName)
	require.NoError(t, err)

	err = nukeAllIamServiceLinkedRoles(session, []*string{&iamServiceLinkedRoleName})
	require.NoError(t, err)

	roleNames, err := getAllIamServiceLinkedRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(roleNames), iamServiceLinkedRoleName)
}

func TestTimeFilterExclusionNewlyCreatedIamServiceLinkedRole(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	// Assert role didn't exist
	name := "cloud-nuke-test-" + util.UniqueID()
	awsServiceName := "autoscaling.amazonaws.com"
	iamServiceLinkedRoleName := "AWSServiceRoleForAutoScaling_" + name

	roleNames, err := getAllIamServiceLinkedRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(roleNames), name)

	// Creates a role
	err = createTestServiceLinkedRole(t, session, name, awsServiceName)
	defer nukeAllIamRoles(session, []*string{&name})

	// Assert role is created
	roleNames, err = getAllIamServiceLinkedRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(roleNames), iamServiceLinkedRoleName)

	// Assert role doesn't appear when we look at roles older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	roleNames, err = getAllIamServiceLinkedRoles(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(roleNames), iamServiceLinkedRoleName)
}

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

func TestListIamRoles(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	roleNames, err := getAllIamRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotEmpty(t, roleNames)
}

func createTestRole(t *testing.T, session *session.Session, name string) error {
	svc := iam.New(session)

	input := &iam.CreateRoleInput{
		RoleName: aws.String(name),
		AssumeRolePolicyDocument: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [
			  {
				"Effect": "Allow",
				"Principal": {
				  "Service": "ec2.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			  }
			]
		  }`),
	}

	_, err := svc.CreateRole(input)
	require.NoError(t, err)

	return nil
}

func createAndAttachInstanceProfile(t *testing.T, session *session.Session, name string) error {
	svc := iam.New(session)

	instanceProfile := &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(name),
	}

	instanceProfileLink := &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String(name),
		RoleName:            aws.String(name),
	}

	_, err := svc.CreateInstanceProfile(instanceProfile)
	require.NoError(t, err)

	_, err = svc.AddRoleToInstanceProfile(instanceProfileLink)
	require.NoError(t, err)

	return nil
}

func TestCreateIamRole(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	name := "cloud-nuke-test-" + util.UniqueID()
	roleNames, err := getAllIamRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(roleNames), name)

	err = createTestRole(t, session, name)
	require.NoError(t, err)

	err = createAndAttachInstanceProfile(t, session, name)
	defer nukeAllIamRoles(session, []*string{&name})
	require.NoError(t, err)

	roleNames, err = getAllIamRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(roleNames), name)
}

func TestNukeIamRoles(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	name := "cloud-nuke-test-" + util.UniqueID()
	err = createTestRole(t, session, name)
	require.NoError(t, err)

	err = createAndAttachInstanceProfile(t, session, name)
	require.NoError(t, err)

	err = nukeAllIamRoles(session, []*string{&name})
	require.NoError(t, err)
}

func TestTimeFilterExclusionNewlyCreatedIamRole(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	// Assert role didn't exist
	name := "cloud-nuke-test-" + util.UniqueID()
	roleNames, err := getAllIamRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(roleNames), name)

	// Creates a role
	err = createTestRole(t, session, name)
	defer nukeAllIamRoles(session, []*string{&name})

	// Assert role is created
	roleNames, err = getAllIamRoles(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(roleNames), name)

	// Assert role doesn't appear when we look at roles older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	roleNames, err = getAllIamRoles(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(roleNames), name)
}

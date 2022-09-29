package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	assert.NotEmpty(t, groupNames) //TODO based on iam test ask if needs preconfiguration to have at least one group
}

//TODO implement create a new testing group, with users?
func createTestGroup(t *testing.T, session *session.Session, name string) error {
	return nil
}
